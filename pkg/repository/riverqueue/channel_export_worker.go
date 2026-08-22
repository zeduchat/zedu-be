package riverqueueBg

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/utility"
)

type ChannelExportWorker struct {
	logger *utility.Logger
	db     *gorm.DB
	river.WorkerDefaults[models.ChannelExportJobArgs]
}

func NewChannelExportWorker(logger *utility.Logger, db *gorm.DB) *ChannelExportWorker {
	return &ChannelExportWorker{
		logger: logger,
		db:     db,
	}
}

func (w *ChannelExportWorker) Work(ctx context.Context, job *river.Job[models.ChannelExportJobArgs]) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Recovered in ChannelExportWorker Work from panic: %v", r)
			debug.PrintStack()
		}
	}()

	exportID := job.Args.ExportID
	channelID := job.Args.ChannelID
	userID := job.Args.UserID
	orgID := job.Args.OrganisationID

	w.logger.Info("Processing ChannelExportWorker job for ExportID: %s, ChannelID: %s, UserID: %s", exportID, channelID, userID)

	var export models.ChannelExport
	if err := w.db.Where("id = ?", exportID).First(&export).Error; err != nil {
		w.logger.Error("Failed to find export record %s: %v", exportID, err)
		return err
	}

	_ = export.UpdateStatus(w.db, models.ExportStatusInProgress, nil, nil, nil)

	if err := w.processExport(ctx, &export, channelID, userID, orgID); err != nil {
		w.logger.Error("Failed channel export %s: %v", exportID, err)
		errMsg := err.Error()
		_ = export.UpdateStatus(w.db, models.ExportStatusFailed, nil, nil, &errMsg)
		return err
	}

	w.logger.Info("Successfully completed ChannelExportWorker job for ExportID: %s", exportID)
	return nil
}

func (w *ChannelExportWorker) processExport(ctx context.Context, export *models.ChannelExport, channelID, userID, orgID string) error {
	var channel models.Channels
	if err := w.db.Where("id = ?", channelID).First(&channel).Error; err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	var exporter struct {
		Username string `json:"username"`
	}
	_ = w.db.Table("users").
		Select("COALESCE(NULLIF(profiles.user_name, ''), users.email) AS username").
		Joins("LEFT JOIN profiles ON profiles.userid = users.id AND (profiles.organisation_id IS NULL OR profiles.organisation_id = ?)", orgID).
		Where("users.id = ?", userID).
		Scan(&exporter).Error

	var allThreads []models.ThreadDocument
	var mediaFilesMap = make(map[string]models.File)
	totalReplyCount := 0

	if storage.DB != nil && storage.DB.Elastic != nil {
		pageSize := 100
		from := 0

		for {
			esQuery := map[string]any{
				"query": map[string]any{
					"bool": map[string]any{
						"must": []map[string]any{
							{
								"term": map[string]any{
									"channels_id": channelID,
								},
							},
							{
								"term": map[string]any{
									"type": "thread",
								},
							},
						},
					},
				},
				"from": from,
				"size": pageSize,
				"sort": []map[string]any{
					{
						"created_at": map[string]any{
							"order": "asc",
						},
					},
				},
			}

			var rawResult any
			err := elastic.SelectAll(storage.DB.Elastic, models.ThreadIndexName, esQuery, &rawResult)
			if err != nil {
				w.logger.Error("Failed to query elasticsearch for channel threads at offset %d: %v", from, err)
				break
			}

			rawMap, ok := rawResult.(map[string]any)
			if !ok {
				break
			}

			hitsObj, ok := rawMap["hits"].(map[string]any)
			if !ok {
				break
			}

			hitsArray, ok := hitsObj["hits"].([]any)
			if !ok || len(hitsArray) == 0 {
				break
			}

			for _, item := range hitsArray {
				hitMap, ok := item.(map[string]any)
				if !ok {
					continue
				}
				sourceMap, ok := hitMap["_source"].(map[string]any)
				if !ok {
					continue
				}

				rawBytes, _ := json.Marshal(sourceMap)
				var doc models.ThreadDocument
				if err := json.Unmarshal(rawBytes, &doc); err == nil {
					if doc.ID == "" {
						if idStr, ok := sourceMap["thread_id"].(string); ok && idStr != "" {
							doc.ID = idStr
						} else if idStr, ok := sourceMap["id"].(string); ok && idStr != "" {
							doc.ID = idStr
						} else if idStr, ok := hitMap["_id"].(string); ok && idStr != "" {
							doc.ID = idStr
						}
					}
					if doc.ChannelsID == "" {
						doc.ChannelsID = channelID
					}
					if doc.OrganisationID == "" {
						doc.OrganisationID = orgID
					}
					allThreads = append(allThreads, doc)
				}
			}

			if len(hitsArray) < pageSize {
				break
			}
			from += pageSize
		}
	}

	var activeThreadIDs []string
	for _, doc := range allThreads {
		if doc.ID != "" {
			activeThreadIDs = append(activeThreadIDs, doc.ID)
		}
	}

	repliesByThreadID := make(map[string][]models.MessageDocument)
	if storage.DB != nil && storage.DB.Elastic != nil && len(activeThreadIDs) > 0 {
		msgFrom := 0
		msgPageSize := 1000

		for {
			msgQuery := map[string]any{
				"query": map[string]any{
					"bool": map[string]any{
						"must": []map[string]any{
							{
								"terms": map[string]any{
									"thread_id": activeThreadIDs,
								},
							},
						},
					},
				},
				"from": msgFrom,
				"size": msgPageSize,
				"sort": []map[string]any{
					{
						"created_at": map[string]any{
							"order": "asc",
						},
					},
				},
			}

			var msgResult any
			err := elastic.SelectAll(storage.DB.Elastic, models.MessageIndexName, msgQuery, &msgResult)
			if err != nil {
				break
			}

			msgs, err := models.UnmarshalMessageResponse(msgResult)
			if err != nil || len(msgs) == 0 {
				break
			}

			for _, reply := range msgs {
				tID := reply.ThreadID.String()
				repliesByThreadID[tID] = append(repliesByThreadID[tID], reply)
			}

			if len(msgs) < msgPageSize {
				break
			}
			msgFrom += msgPageSize
		}
	}

	for i := range allThreads {
		tID := allThreads[i].ID
		if fullReplies, found := repliesByThreadID[tID]; found && len(fullReplies) > 0 {
			allThreads[i].Messages = fullReplies
		}

		totalReplyCount += len(allThreads[i].Messages)

		for _, mf := range allThreads[i].Media {
			if mf.ID != "" {
				mediaFilesMap[mf.ID] = mf
			}
		}
		for _, reply := range allThreads[i].Messages {
			for _, rmf := range reply.Media {
				if rmf.ID != "" {
					mediaFilesMap[rmf.ID] = rmf
				}
			}
		}
	}

	w.logger.Info("[channel export] Processed %d threads and %d total replies for export %s", len(allThreads), totalReplyCount, export.ID)

	var dbMediaFiles []models.File
	_ = w.db.Where("channel_id = ? AND file_type != 'zip' AND file_name NOT LIKE 'export_%' AND deleted_at IS NULL", channelID).Find(&dbMediaFiles).Error
	for _, mf := range dbMediaFiles {
		if mf.ID != "" {
			mediaFilesMap[mf.ID] = mf
		}
	}

	var missingMediaIDs []string
	for mID, mf := range mediaFilesMap {
		if mf.FileLink == "" || mf.FileName == "" {
			missingMediaIDs = append(missingMediaIDs, mID)
		}
	}
	if len(missingMediaIDs) > 0 {
		var fetchedFiles []models.File
		if err := w.db.Where("id IN ?", missingMediaIDs).Find(&fetchedFiles).Error; err == nil {
			for _, ff := range fetchedFiles {
				if ff.ID != "" {
					mediaFilesMap[ff.ID] = ff
				}
			}
		}
	}

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	w.logger.Info("[channel export] Creating channel info and threads JSON metadata in export archive %s", export.ID)

	channelMetaData := map[string]interface{}{
		"channel_id":           channel.ID,
		"name":                 channel.Name,
		"description":          channel.Description,
		"topic":                channel.Topic,
		"organisation_id":      channel.OrganisationID,
		"exported_by":          userID,
		"exported_by_username": exporter.Username,
		"exported_at":          time.Now().Format(time.RFC3339),
		"total_threads":        len(allThreads),
		"total_replies":        totalReplyCount,
		"total_media":          len(mediaFilesMap),
	}
	channelMetaBytes, _ := json.MarshalIndent(channelMetaData, "", "  ")
	fMeta, err := zipWriter.Create("channel_info.json")
	if err == nil {
		_, _ = fMeta.Write(channelMetaBytes)
	}

	threadsBytes, _ := json.MarshalIndent(allThreads, "", "  ")
	fMsgs, err := zipWriter.Create("threads.json")
	if err == nil {
		_, _ = fMsgs.Write(threadsBytes)
	}

	w.logger.Info("[channel export] Creating and archiving %d media files for export %s", len(mediaFilesMap), export.ID)

	if storage.DB != nil && storage.DB.Minio != nil && len(mediaFilesMap) > 0 {
		minioClient := storage.DB.Minio
		bucketName := config.Config.Minio.BucketName

		for _, mf := range mediaFilesMap {
			if mf.FileLink == "" {
				continue
			}
			var objectKey string
			if idx := strings.Index(mf.FileLink, bucketName+"/"); idx != -1 {
				objectKey = mf.FileLink[idx+len(bucketName)+1:]
			} else {
				objectKey = utility.ExtractHashedFileName(mf.FileLink)
				if objectKey != "" {
					objectKey = "public/file-uploads/" + objectKey
				}
			}

			if objectKey == "" {
				continue
			}

			stat, statErr := minioClient.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
			if statErr != nil || stat.Size == 0 {
				w.logger.Warning("Skipping media file %s (key %s) stat error or empty: %v", mf.ID, objectKey, statErr)
				continue
			}

			obj, err := minioClient.GetObject(ctx, bucketName, objectKey, minio.GetObjectOptions{})
			if err != nil {
				continue
			}

			idPrefix := mf.ID
			if len(idPrefix) > 8 {
				idPrefix = idPrefix[:8]
			}
			fileNameInZip := fmt.Sprintf("media/%s_%s", idPrefix, mf.FileName)
			fMedia, err := zipWriter.Create(fileNameInZip)
			if err == nil {
				_, _ = io.Copy(fMedia, obj)
			}
			_ = obj.Close()
		}
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize zip archive: %w", err)
	}

	if storage.DB == nil || storage.DB.Minio == nil {
		return fmt.Errorf("minio client not initialized")
	}

	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName
	zipBytes := zipBuffer.Bytes()
	zipSize := int64(len(zipBytes))

	sanitizedChannelName := strings.ReplaceAll(channel.Name, " ", "_")
	zipFileName := fmt.Sprintf("export_%s_%d.zip", sanitizedChannelName, time.Now().Unix())
	encodedFilePath := "public/file-uploads/exports/" + zipFileName

	_, err = minioClient.PutObject(ctx, bucketName, encodedFilePath, bytes.NewReader(zipBytes), zipSize, minio.PutObjectOptions{
		ContentType: "application/zip",
	})
	if err != nil {
		return fmt.Errorf("failed to upload export zip to minio: %w", err)
	}

	fileURL := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

	now := time.Now()
	fileRecord := models.File{
		ID:             utility.GenerateUUID(),
		FileName:       zipFileName,
		FileType:       "zip",
		MimeType:       "application/zip",
		FileLink:       fileURL,
		Size:           zipSize,
		OrganisationID: orgID,
		UserID:         userID,
		ChannelID:      &channelID,
		LastAccessedAt: &now,
	}

	if err := fileRecord.CreateFileRecord(w.db); err != nil {
		return fmt.Errorf("failed to create file record in file manager: %w", err)
	}

	return export.UpdateStatus(w.db, models.ExportStatusCompleted, &fileRecord.ID, &fileURL, nil)
}

var _ = os.MkdirAll
var _ = filepath.Base
