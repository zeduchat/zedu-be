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
	"regexp"
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

const maxFileSize = 200 * 1024 * 1024

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

func UploadFiles(db *gorm.DB, logger *utility.Logger, file multipart.File, header *multipart.FileHeader, folderID, orgID, userID string) (*models.File, error) {
	var generatedUrl string
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	if header.Size > maxFileSize {
		errMsg := "file exceeds max size"
		utility.LogAndPrint(logger, errMsg)
		return nil, fmt.Errorf("file exceeds max size")
	}

	fileHash, hashErr := HashFile(file)
	if hashErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to hash file: %v", hashErr))
		return nil, fmt.Errorf("failed to hash file: %v", hashErr)
	}

	if seeker, ok := file.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("failed to seek file: %v", err))
			return nil, fmt.Errorf("failed to seek file: %v", err)
		}
	}

	mimeType, mimeTypeErr := DetectMimeType(file)
	if mimeTypeErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("could not detect file type: %v", mimeTypeErr.Error()))
		return nil, fmt.Errorf("could not detect file type: %w", mimeTypeErr)
	}

	extension := filepath.Ext(header.Filename)
	hashedFileName := fmt.Sprintf("%s%s", fileHash, extension)
	encodedFilePath := "public/file-uploads/" + hashedFileName

	exists, existsErr := FileExists(logger, encodedFilePath)
	if existsErr != nil && !exists {
		utility.LogAndPrint(logger, fmt.Sprintf("error: %v.", existsErr.Error()))
		return nil, existsErr
	} else if exists {
		utility.LogAndPrint(logger, "checking for file existence")
		existingFileURL := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

		var existingFile models.File
		err := db.Where("file_link = ?", existingFileURL).First(&existingFile).Error
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("File exists in Minio bucket. Failed to retrieve existing metadata in database: %v", err.Error()))
			return nil, fmt.Errorf("file exists in Minio bucket. failed to retrieve existing metadata in database: %v", err)
		}

		// if file exists, we create a new record for the user/org pointing to the same physical file
		utility.LogAndPrint(logger, "File exists, creating a new DB entry for current user or org")

		var fID *string
		if folderID != "" {
			fID = &folderID
		}

		newFileEntry := models.File{
			ID:             utility.GenerateUUID(),
			FileName:       header.Filename,
			FileType:       filepath.Ext(header.Filename)[1:],
			MimeType:       mimeType,
			FileLink:       existingFile.FileLink,
			Size:           header.Size,
			OrganisationID: orgID,
			UserID:         userID,
			FolderID:       fID,
		}
		storageErr := newFileEntry.CreateFileRecord(db)
		if storageErr != nil {
			errMsg := fmt.Errorf("error saving new file details: %w", storageErr)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to save new file details to database: %v", errMsg))
			return nil, errMsg
		}
		return &newFileEntry, nil

	} else {
		_, err := minioClient.PutObject(context.Background(), bucketName, encodedFilePath, file, header.Size, minio.PutObjectOptions{ContentType: mimeType})
		if err != nil {
			errMsg := fmt.Errorf("failed to upload file to %s: %w", encodedFilePath, err)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", encodedFilePath, err.Error()))
			return nil, errMsg
		}

		(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", encodedFilePath))
		generatedUrl = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

		var fID *string
		if folderID != "" {
			fID = &folderID
		}

		response := models.File{
			ID:             utility.GenerateUUID(),
			FileName:       header.Filename,
			FileType:       filepath.Ext(header.Filename)[1:],
			MimeType:       mimeType,
			FileLink:       generatedUrl,
			Size:           header.Size,
			OrganisationID: orgID,
			UserID:         userID,
			FolderID:       fID,
		}

		storageErr := response.CreateFileRecord(db)
		if storageErr != nil {
			errMsg := fmt.Errorf("error saving file details: %w", storageErr)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to save file details to database: %v", errMsg.Error()))
			return nil, errMsg
		}

		return &response, nil
	}
}

func CreateFolder(db *gorm.DB, name, orgID, userID string, parentID *string) (*models.Folder, error) {
	folder := models.Folder{
		ID:             utility.GenerateUUID(),
		Name:           name,
		OrganisationID: orgID,
		UserID:         userID,
		ParentID:       parentID,
	}
	err := postgresql.CreateOneRecord(db, &folder)
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

func GetFolders(db *gorm.DB, orgID string, parentID *string) ([]models.Folder, error) {
	var folders []models.Folder
	query := db.Where("organisation_id = ?", orgID)
	if parentID != nil {
		query = query.Where("parent_id = ?", parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	err := query.Find(&folders).Error
	return folders, err
}

func DeleteFolder(db *gorm.DB, folderID string) error {
	// soft delete folder
	err := db.Where("id = ?", folderID).Delete(&models.Folder{}).Error
	if err != nil {
		return err
	}

	// soft delete the files in it too
	err = db.Where("folder_id = ?", folderID).Delete(&models.File{}).Error
	return err
}

func UpdateFolder(db *gorm.DB, folderID, name string) (*models.Folder, error) {
	var folder models.Folder
	err := db.Where("id = ?", folderID).First(&folder).Error
	if err != nil {
		return nil, err
	}
	folder.Name = name
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

func DeleteFileDetailsByID(logger *utility.Logger, db *storage.Database, file *models.File, fileId, threadId string) error {
	var fileModel models.File

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

	// soft delete the record
	err := fileModel.DeleteFileByID(db.Postgresql, fileId)
	if err != nil {
		return err
	}

	if threadId != "" {
		err = RemoveMediaFileFromThread(db.Elastic, threadId, fileId, logger)
		if err != nil {
			logger.Error("Failed to remove media file from thread, err: %s", err)
		}
	}

	return nil
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
func UpdateFileName(db *gorm.DB, fileId, newFileName, orgID, userID string, logger *utility.Logger) (*models.File, error) {
	trimmed, err := Validate(newFileName)
	if err != nil {
		return nil, err
	}

	fileResponse, err := GetFileDetailsByID(db, fileId)
	if err != nil {
		return nil, err
	}
	if fileResponse == nil {
		return nil, fmt.Errorf("file does not exist")
	}
	err = fileResponse.UpdateFileName(db, fileId, trimmed)
	if err != nil {
		return nil, err
	}
	fileResponse, err = GetFileDetailsByID(db, fileId)
	if err != nil {
		return nil, err
	}
	notification := models.Notification[models.UpdatedMedia]
	notification.SectionType = models.ThreadSection
	notification.Content = fileResponse
	notification.ModificationDetails = &models.ModificationDetails{
		UserId: userID,
		OrgId:  orgID,
	}
	userChannelID := fmt.Sprintf("%s/%s", orgID, userID)
	if err := centrifuge.PublishChannel(logger, userChannelID, notification); err != nil {
		logger.Error("Error Publishing notification event: %v", err)
	}

	return fileResponse, nil
}

// method for the validation heavy lifting.
func Validate(filename string) (string, error) {
	trimmed := strings.TrimSpace(filename)
	if strings.HasPrefix(trimmed, ".") {
		return "", fmt.Errorf("filename cannot start with a period")
	}
	if trimmed == "" {
		return "", fmt.Errorf("file name cannot be empty")
	}
	if len(trimmed) > 255 {
		return "", fmt.Errorf("file name too long")
	}
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\s._-]+$`)
	if !validPattern.MatchString(trimmed) {
		return "", fmt.Errorf("filename contains invalid characters")
	}
	return trimmed, nil
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

func GetFiles(db *gorm.DB, orgID, userID string, queryParams map[string]string, page, limit int) ([]models.File, int64, error) {
	var files []models.File
	var count int64

	offset := (page - 1) * limit

	query := db.Model(&models.File{}).
		Select("files.*, profiles.full_name as owner_name, profiles.avatar_url as owner_avatar").
		Joins("LEFT JOIN profiles ON profiles.userid = files.user_id").
		Where("files.organisation_id = ?", orgID)

	// mode options: all, mine, shared, trash
	mode := queryParams["mode"]

	switch mode {
	case "mine":
		query = query.Where("files.user_id = ?", userID)
	case "shared":
		// Get channels the user is in
		var userChannels []string
		err := db.Model(&models.UserChannels{}).Where("user_id = ?", userID).Pluck("channels_id", &userChannels).Error
		if err != nil {
			return nil, 0, err
		}
		if len(userChannels) > 0 {
			query = query.Where("files.channel_id IN ?", userChannels)
		} else {
			// User is in no channels, return empty or handle as needed.
			// For now, returning files where channel_id is null might be wrong if we strictly want shared files.
			// Let's assume shared means "in a channel I have access to".
			// If no channels, then no shared files.
			return []models.File{}, 0, nil
		}
	case "trash":
		query = query.Unscoped().Where("files.deleted_at IS NOT NULL")
	default:
		// "all" or default: view all files in org (that are not deleted)
		// Usually "all" might imply "all files I have access to".
		// For simplicity/admin view, we keep org-wide.
		// But if we want to restrict to "public" or "my channels", logic would be more complex.
		// Current existing logic was just orgID. Keeping it consistent but adding deleted_at check.
		query = query.Where("files.deleted_at IS NULL")
	}

	if folderID, ok := queryParams["folder_id"]; ok && folderID != "" {
		query = query.Where("files.folder_id = ?", folderID)
	} else if mode != "trash" && mode != "search" && mode != "shared" {
		// Only filter by root folder if not searching, not in trash, and not in shared mode
		// Shared mode files might be in folders or not, but usually we list them as a flat list or by channel.
		// For now, let's not restrict by folder_id IS NULL for shared mode unless explicitly asked.
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

	err := query.Count(&count).Offset(offset).Limit(limit).Find(&files).Error
	return files, count, err
}
