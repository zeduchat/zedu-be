package fileManagement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func FormatFileName(filename string) string {
	return strings.ReplaceAll(filename, " ", "_")
}

func DetectMimeType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return "", err
	}
	file.Seek(0, 0)
	mimeType := http.DetectContentType(buffer)
	return mimeType, nil
}

func GetMimeTypeFromFileName(filename string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "", fmt.Errorf("filename has no extension")
	}

	mimeType := mime.TypeByExtension(strings.ToLower(ext))
	if mimeType == "" {
		return "application/octet-stream", nil
	}

	return mimeType, nil
}

func HashFile(file multipart.File) (string, error) {
	hasher := sha256.New()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func FileExists(logger *utility.Logger, fileName string) (bool, error) {
	minioClient := storage.DB.Minio

	bucketName := config.Config.Minio.BucketName

	_, err := minioClient.StatObject(context.Background(), bucketName, fileName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			utility.LogAndPrint(logger, fmt.Sprintf("File %s does not exist in bucket %s", fileName, bucketName))
			return false, nil
		}
		utility.LogAndPrint(logger, fmt.Sprintf("Error checking if file %s exists: %v", fileName, err))
		return false, fmt.Errorf("error checking if file %s exists: %v", fileName, err)
	}

	utility.LogAndPrint(logger, fmt.Sprintf("File %s exists in bucket %s", fileName, bucketName))
	return true, fmt.Errorf("file %s exists in bucket %s", fileName, bucketName)
}

func UploadFile(db *gorm.DB, logger *utility.Logger, params models.UploadFileParams) (*models.File, error) {
	var generatedUrl string
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	var fID *string
	if params.FolderID != "" {
		fID = &params.FolderID
	}

	fileHash, hashErr := HashFile(params.File)
	if hashErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to hash file: %v", hashErr))
		return nil, fmt.Errorf("failed to hash file: %v", hashErr)
	}

	if seeker, ok := params.File.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("failed to seek file: %v", err))
			return nil, fmt.Errorf("failed to seek file: %v", err)
		}
	}

	mimeType, mimeTypeErr := DetectMimeType(params.File)
	if mimeTypeErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("could not detect file type: %v", mimeTypeErr.Error()))
		return nil, fmt.Errorf("could not detect file type: %w", mimeTypeErr)
	}

	extension := filepath.Ext(params.Header.Filename)
	hashedFileName := fmt.Sprintf("%s%s", fileHash, extension)
	encodedFilePath := "public/file-uploads/" + hashedFileName

	exists, existsErr := FileExists(logger, encodedFilePath)
	if existsErr != nil && !exists {
		utility.LogAndPrint(logger, fmt.Sprintf("error: %v.", existsErr.Error()))
		return nil, existsErr
	}

	if exists {
		utility.LogAndPrint(logger, "checking for file existence")
		existingFileURL := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

		var existingFile models.File
		err := db.Where("file_link = ?", existingFileURL).First(&existingFile).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				utility.LogAndPrint(logger, fmt.Sprintf("File exists in Minio bucket but not in DB. Creating new record for: %s", existingFileURL))

				// persists file details and skip MinIO upload since it's already there
				saveParams := CreateAndSaveFileParams{
					UploadParams: params,
					FileLink:     existingFileURL,
					MimeType:     mimeType,
					FolderID:     fID,
				}

				newFileEntry, storageErr := createAndSaveFile(db, saveParams)
				if storageErr != nil {
					errMsg := fmt.Errorf("error saving new file details: %w", storageErr)
					utility.LogAndPrint(logger, fmt.Sprintf("failed to save new file details to database: %v", errMsg))
					return nil, errMsg
				}

				return newFileEntry, nil
			}

			utility.LogAndPrint(logger, fmt.Sprintf("File exists in Minio bucket. Failed to retrieve existing metadata in database: %v", err.Error()))
			return nil, fmt.Errorf("file exists in Minio bucket. failed to retrieve existing metadata in database: %v", err)
		}

		// if file exists, we create a new record for the user/org pointing to the same physical file
		utility.LogAndPrint(logger, "File exists, creating a new DB entry for current user or org")

		// check if a file record with the same metadata already exists
		var duplicateFile models.File
		query := db.Where("file_name = ? AND organisation_id = ? AND user_id = ? AND file_link = ?", params.Header.Filename, params.OrgID, params.UserID, existingFile.FileLink)

		if fID != nil {
			query = query.Where("folder_id = ?", *fID)
		} else {
			query = query.Where("folder_id IS NULL")
		}

		if err := query.First(&duplicateFile).Error; err == nil {
			utility.LogAndPrint(logger, "File record already exists, returning existing record")
			return &duplicateFile, nil
		}

		saveParams := CreateAndSaveFileParams{
			UploadParams: params,
			FileLink:     existingFile.FileLink,
			MimeType:     mimeType,
			FolderID:     fID,
		}
		newFileEntry, storageErr := createAndSaveFile(db, saveParams)

		if storageErr != nil {
			errMsg := fmt.Errorf("error saving new file details: %w", storageErr)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to save new file details to database: %v", errMsg))
			return nil, errMsg
		}

		return newFileEntry, nil

	}

	_, err := minioClient.PutObject(context.Background(), bucketName, encodedFilePath, params.File, params.Header.Size, minio.PutObjectOptions{ContentType: mimeType})
	if err != nil {
		errMsg := fmt.Errorf("failed to upload file to %s: %w", encodedFilePath, err)
		utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", encodedFilePath, err.Error()))
		return nil, errMsg
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", encodedFilePath))
	generatedUrl = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

	saveParams := CreateAndSaveFileParams{
		UploadParams: params,
		FileLink:     generatedUrl,
		MimeType:     mimeType,
		FolderID:     fID,
	}
	response, storageErr := createAndSaveFile(db, saveParams)
	if storageErr != nil {
		// rollback file from MinIO if DB save fails
		utility.LogAndPrint(logger, "Failed to save file details to DB, rolling back MinIO upload...")
		removeErr := minioClient.RemoveObject(context.Background(), bucketName, encodedFilePath, minio.RemoveObjectOptions{})
		if removeErr != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Failed to rollback MinIO upload for %s: %v", encodedFilePath, removeErr))
		} else {
			utility.LogAndPrint(logger, fmt.Sprintf("Successfully rolled back MinIO upload for %s", encodedFilePath))
		}

		errMsg := fmt.Errorf("error saving file details: %w", storageErr)
		utility.LogAndPrint(logger, fmt.Sprintf("failed to save file details to database: %v", errMsg.Error()))
		return nil, errMsg
	}

	return response, nil

}

type CreateAndSaveFileParams struct {
	UploadParams models.UploadFileParams
	FileLink     string
	MimeType     string
	FolderID     *string
}

func createAndSaveFile(db *gorm.DB, params CreateAndSaveFileParams) (*models.File, error) {
	newFileEntry := models.File{
		ID:             utility.GenerateUUID(),
		FileName:       params.UploadParams.Header.Filename,
		FileType:       filepath.Ext(params.UploadParams.Header.Filename)[1:],
		MimeType:       params.MimeType,
		FileLink:       params.FileLink,
		Size:           params.UploadParams.Header.Size,
		OrganisationID: params.UploadParams.OrgID,
		UserID:         params.UploadParams.UserID,
		FolderID:       params.FolderID,
	}

	if err := newFileEntry.CreateFileRecord(db); err != nil {
		return nil, err
	}
	return &newFileEntry, nil
}

func CreateFolder(db *gorm.DB, params models.CreateFolderParams) (*models.Folder, error) {
	folder := models.Folder{
		ID:             utility.GenerateUUID(),
		Name:           params.Name,
		OrganisationID: params.OrganisationID,
		UserID:         params.UserID,
	}
	err := postgresql.CreateOneRecord(db, &folder)
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func GetFolders(db *gorm.DB, orgID string) ([]models.Folder, error) {
	var folders []models.Folder
	query := db.Where("organisation_id = ?", orgID)

	err := query.Find(&folders).Error
	return folders, err
}

func DeleteFolder(db *gorm.DB, folderID string, permanent bool) error {
	if permanent {
		// hard delete folder and its files
		err := db.Unscoped().Where("id = ?", folderID).Delete(&models.Folder{}).Error
		if err != nil {
			return err
		}

		return db.Unscoped().Where("folder_id = ?", folderID).Delete(&models.File{}).Error
	}

	// soft delete folder
	err := db.Where("id = ?", folderID).Delete(&models.Folder{}).Error
	if err != nil {
		return err
	}

	// soft delete the files in it too
	return db.Where("folder_id = ?", folderID).Delete(&models.File{}).Error
}

func UpdateFolder(db *gorm.DB, params models.UpdateFolderParams) (*models.Folder, error) {
	var folder models.Folder
	err := db.Where("id = ? AND organisation_id = ?", params.FolderID, params.OrgID).First(&folder).Error
	if err != nil {
		return nil, err
	}

	// check permissions
	if folder.UserID != params.UserID {
		return nil, fmt.Errorf("unauthorized to update this folder")
	}

	if folder.Name == params.Name {
		return &folder, nil
	}

	folder.Name = params.Name
	err = db.Save(&folder).Error
	return &folder, err
}

func GetFileDetailsByID(db *gorm.DB, fileId string) (*models.File, error) {
	var fileModel models.File

	file, err := fileModel.GetFileByID(db, fileId)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func GetFileDetailsByIDUnscoped(db *gorm.DB, fileId string) (*models.File, error) {
	var file models.File
	err := db.Unscoped().Where("id = ?", fileId).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func DeleteFileDetailsByID(logger *utility.Logger, db *storage.Database, file *models.File, fileId, threadId string, permanent bool) error {
	var fileModel models.File

	if permanent {

		// check if other records point to the same physical file
		count, countErr := fileModel.GetFileCountByLink(db.Postgresql, file.FileLink)
		if countErr != nil {
			return countErr
		}

		// if this is the last record pointing to the file, delete from MinIO
		if count == 1 {
			hashedFileName := utility.ExtractHashedFileName(file.FileLink)

			minioErr := models.DeleteUploadedFiles(logger, hashedFileName)
			if minioErr != nil {
				return minioErr
			}
		}

		// hard delete the record
		err := db.Postgresql.Unscoped().Where("id = ?", fileId).Delete(&models.File{}).Error
		if err != nil {
			return err
		}
	} else {

		// soft delete the record
		err := fileModel.DeleteFileByID(db.Postgresql, fileId)
		if err != nil {
			return err
		}
	}

	if threadId != "" {
		err := RemoveMediaFileFromThread(db.Elastic, threadId, fileId, logger)
		if err != nil {
			logger.Error("Failed to remove media file from thread, err: %s", err)
		}
	}

	return nil
}

func DeleteMultipleFiles(logger *utility.Logger, db *storage.Database, fileIDs []string, permanent bool) (int, int, []error) {
	successCount := 0
	failureCount := 0
	var errs []error

	for _, fileID := range fileIDs {
		var file *models.File
		var err error

		if permanent {
			file, err = GetFileDetailsByIDUnscoped(db.Postgresql, fileID)
		} else {
			file, err = GetFileDetailsByID(db.Postgresql, fileID)
		}

		if err != nil {
			failureCount++
			errs = append(errs, fmt.Errorf("failed to find file %s: %v", fileID, err))
			continue
		}

		err = DeleteFileDetailsByID(logger, db, file, fileID, "", permanent)
		if err != nil {
			failureCount++
			errs = append(errs, fmt.Errorf("failed to delete file %s: %v", fileID, err))
		} else {
			successCount++
		}
	}

	return successCount, failureCount, errs
}

func DeleteMultipleFolders(db *gorm.DB, folderIDs []string, permanent bool) (int, int, []error) {
	successCount := 0
	failureCount := 0
	var errs []error

	for _, folderID := range folderIDs {
		err := DeleteFolder(db, folderID, permanent)
		if err != nil {
			failureCount++
			errs = append(errs, fmt.Errorf("failed to delete folder %s: %v", folderID, err))
		} else {
			successCount++
		}
	}

	return successCount, failureCount, errs
}

func RemoveMediaFileFromThread(db *elasticsearch.Client, threadID, fileID string, logger *utility.Logger) error {

	var (
		threadDoc models.ThreadDocument
	)

	err := threadDoc.GetThreadById(threadID)
	if err != nil {
		return errors.New("thread not found")
	}

	script := `if (ctx._source.media == null) {
		ctx._source.media = [];
	}
	for (int i = 0; i < ctx._source.media.size(); i++) {
		if (ctx._source.media[i] != null && ctx._source.media[i].id == params.file_id) {
			ctx._source.media.remove(i);
			break;
		}
	}`

	req := map[string]any{
		"script": map[string]any{
			"source": script,
			"params": map[string]any{
				"file_id": fileID,
			},
		},
	}

	err = elastic.UpdateDocWithScript(db, models.ThreadIndexName, threadID, req)

	if err != nil {
		return fmt.Errorf("failed to remove media file from thread: %w", err)
	}

	err = threadDoc.GetThreadById(threadID)
	if err != nil {
		return errors.New("thread not found")
	}

	notification := models.Notification[models.UpdatedMedia]
	notification.SectionType = models.ThreadSection
	notification.Content = threadDoc
	notification.ModificationDetails = &models.ModificationDetails{
		ThreadId:  threadID,
		ChannelId: threadDoc.ChannelsID,
	}

	if err := centrifuge.PublishChannel(logger, threadDoc.ChannelsID, notification); err != nil {
		logger.Error("Error Publishing to with destination id: %s error: %v", threadDoc.ChannelsID, err)
		return errors.New("failed to publish data")
	}

	return nil
}

// method to validate and update  filename.
func UpdateFileName(db *gorm.DB, logger *utility.Logger, params models.UpdateFileNameParams) (*models.File, error) {

	fileResponse, err := GetFileDetailsByID(db, params.FileID)
	if err != nil {
		return nil, err
	}
	if fileResponse == nil {
		return nil, fmt.Errorf("file does not exist")
	}
	err = fileResponse.UpdateFileName(db, params.FileID, params.NewFileName)
	if err != nil {
		return nil, err
	}
	// fileResponse is now updated with the new name
	notification := models.Notification[models.UpdatedMedia]
	notification.SectionType = models.ThreadSection
	notification.Content = fileResponse
	notification.ModificationDetails = &models.ModificationDetails{
		UserId: params.UserID,
		OrgId:  params.OrgID,
	}
	userChannelID := fmt.Sprintf("%s/%s", params.OrgID, params.UserID)
	if err := centrifuge.PublishChannel(logger, userChannelID, notification); err != nil {
		logger.Error("Error Publishing notification event: %v", err)
	}

	return fileResponse, nil
}

func MoveFile(db *gorm.DB, fileID, folderID string) (*models.File, error) {
	var file models.File
	err := db.Where("id = ?", fileID).First(&file).Error
	if err != nil {
		return nil, err
	}

	var fID *string
	if folderID != "" {
		fID = &folderID
	} else {
		fID = nil // move to root
	}

	file.FolderID = fID
	err = db.Save(&file).Error
	return &file, err
}

func GetFiles(db *gorm.DB, params models.GetFilesParams) ([]models.File, int64, error) {
	var files []models.File
	var count int64

	offset := (params.Page - 1) * params.Limit
	queryParams := params.QueryParams

	query := db.Model(&models.File{}).
		Select("files.*, profiles.full_name as owner_name, profiles.avatar_url as owner_avatar").
		Joins("LEFT JOIN profiles ON profiles.userid = files.user_id").
		Where("files.organisation_id = ?", params.OrgID)

	// mode options: all, mine, shared, trash
	mode := queryParams["mode"]

	switch mode {
	case "mine":
		query = query.Where("files.user_id = ?", params.UserID)
	case "shared":
		// Get channels the user is in will check for messageid for dm files when it'll be implemented

		var userChannels []string
		err := db.Model(&models.UserChannels{}).Where("user_id = ?", params.UserID).Pluck("channels_id", &userChannels).Error
		if err != nil {
			return nil, 0, err
		}

		if len(userChannels) > 0 {
			query = query.Where("files.channel_id IN ?", userChannels)
		} else {

			return []models.File{}, 0, nil
		}
	case "trash":
		query = query.Unscoped().Where("files.deleted_at IS NOT NULL")
	default:
		// "all" or default: view all files in org (that are not deleted)

		query = query.Where("files.deleted_at IS NULL")
	}

	if folderID, ok := queryParams["folder_id"]; ok && folderID != "" {
		query = query.Where("files.folder_id = ?", folderID)
	} else if mode != "trash" && mode != "search" && mode != "shared" {

		if _, searching := queryParams["search"]; !searching {
			query = query.Where("files.folder_id IS NULL")
		}
	}

	if search, ok := queryParams["search"]; ok && search != "" {
		query = query.Where("files.file_name ILIKE ?", "%"+search+"%")
	}

	if fileType, ok := queryParams["type"]; ok && fileType != "" {
		query = query.Where("files.file_type = ?", fileType)
	}

	if fileCategory, ok := queryParams["file_category"]; ok && fileCategory != "" {
		switch strings.ToLower(fileCategory) {
		case "documents":
			query = query.Where(`(
				files.mime_type LIKE '%pdf%' OR
				files.mime_type LIKE '%document%' OR
				files.mime_type LIKE '%word%' OR
				files.mime_type LIKE '%text%' OR
				files.mime_type LIKE '%rtf%' OR
				files.mime_type LIKE '%doc' OR
				files.mime_type LIKE '%docx' OR
				files.mime_type LIKE '%txt' OR
				files.mime_type LIKE '%odt'
			)`)
		case "spreadsheets":
			query = query.Where(`(
				files.mime_type LIKE '%spreadsheet%' OR
				files.mime_type LIKE '%excel%' OR
				files.mime_type LIKE '%xls' OR
				files.mime_type LIKE '%xlsx' OR
				files.mime_type LIKE '%csv' OR
				files.mime_type LIKE '%ods'
			)`)
		case "images":
			query = query.Where("files.mime_type LIKE 'image/%'")
		case "videos":
			query = query.Where("files.mime_type LIKE 'video/%'")
		case "music":
			query = query.Where("files.mime_type LIKE 'audio/%'")
		}
	}

	if dateFilter, ok := queryParams["date_modified"]; ok && dateFilter != "" {
		startDate, endDate := models.GetDateRangeFilter(dateFilter)
		if startDate != nil && endDate != nil {
			query = query.Where("files.updated_at BETWEEN ? AND ?", startDate, endDate)
		}
	}

	err := query.Count(&count).Offset(offset).Limit(params.Limit).Find(&files).Error
	return files, count, err
}

func GetFileInfo(db *gorm.DB, params models.File) *models.FileInfoResponse {
	sharedIn := []string{}
	if params.ChannelID != nil {
		var channel models.Channels
		err := db.Select("name").Where("id = ?", *params.ChannelID).First(&channel).Error
		if err == nil {
			sharedIn = append(sharedIn, channel.Name)
		}
	}

	// Fetch username from profile
	var profile models.Profile
	owner := params.UserID // fallback to user ID
	err := db.Select("user_name").Where("userid = ?", params.UserID).First(&profile).Error
	if err == nil && profile.UserName != "" {
		owner = profile.UserName
	}

	response := &models.FileInfoResponse{
		Owner:        owner,
		DateUploaded: params.CreatedAt,
		LastUpdated:  params.UpdatedAt,
		SharedIn:     sharedIn,
	}
	return response

}

var (
	ErrFileNotDeleted      = errors.New("file is not deleted")
	ErrUnauthorizedRestore = errors.New("unauthorized to restore this file")
)

func RestoreFile(db *gorm.DB, fileID string, userID string) (*models.File, error) {
	var file models.File

	// use Unscoped to find soft deleted records efficiently
	err := db.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", fileID).First(&file).Error
	if err != nil {
		return nil, err
	}

	// check permissions
	if file.UserID != userID {
		return nil, ErrUnauthorizedRestore
	}

	// restore the file by clearing [deleted_at]
	// avoid hooks if any, and set [deleted_at] to NULL
	err = db.Model(&file).Unscoped().UpdateColumn("deleted_at", nil).Error
	if err != nil {
		return nil, err
	}

	return &file, nil
}
