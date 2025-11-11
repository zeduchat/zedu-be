package fileManagement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
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

	// Copy file content into the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	// Convert hash to a hexadecimal string
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

func UploadFiles(db *gorm.DB, logger *utility.Logger, file multipart.File, header *multipart.FileHeader) (*models.UploadedFileResponse, error) {
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

		var existingFile models.UploadedFileResponse
		err := db.Where("file_link = ?", existingFileURL).First(&existingFile).Error
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("File exists in Minio bucket. Failed to retrieve existing metadata in database: %v", err.Error()))
			return nil, fmt.Errorf("file exists in Minio bucket. failed to retrieve existing metadata in database: %v", err)
		}

		if existingFile.FileName == header.Filename {
			utility.LogAndPrint(logger, "Using existing file reference with the same name")
			return &existingFile, nil
		} else {
			utility.LogAndPrint(logger, "File exists but with a different name, creating a new DB entry")
			newFileEntry := models.UploadedFileResponse{
				ID:       utility.GenerateUUID(),
				FileName: header.Filename,
				FileType: filepath.Ext(header.Filename)[1:],
				MimeType: mimeType,
				FileLink: existingFile.FileLink,
			}
			storageErr := newFileEntry.CreateFileRecord(db)
			if storageErr != nil {
				errMsg := fmt.Errorf("error saving new file details: %w", storageErr)
				utility.LogAndPrint(logger, fmt.Sprintf("failed to save new file details to database: %v", errMsg))
				return nil, errMsg
			}
			return &newFileEntry, nil
		}

	} else {
		_, err := minioClient.PutObject(context.Background(), bucketName, encodedFilePath, file, header.Size, minio.PutObjectOptions{ContentType: mimeType})
		if err != nil {
			errMsg := fmt.Errorf("failed to upload file to %s: %w", encodedFilePath, err)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", encodedFilePath, err.Error()))
			return nil, errMsg
		}

		(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", encodedFilePath))
		generatedUrl = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

		response := models.UploadedFileResponse{
			ID:       utility.GenerateUUID(),
			FileName: header.Filename,
			FileType: filepath.Ext(header.Filename)[1:],
			MimeType: mimeType,
			FileLink: generatedUrl,
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

func GetFileDetailsByID(db *gorm.DB, fileId string) (*models.UploadedFileResponse, error) {
	var fileModel models.UploadedFileResponse

	file, err := fileModel.GetFileByID(db, fileId)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func DeleteFileDetailsByID(logger *utility.Logger, db *storage.Database, file *models.UploadedFileResponse, fileId, threadId string) error {
	var fileModel models.UploadedFileResponse

	count, countErr := fileModel.GetFileCountByLink(db.Postgresql, file.FileLink)
	if countErr != nil {
		return countErr
	}
	if count == 1 {
		hashedFileName := utility.ExtractHashedFileName(file.FileLink)

		minioErr := models.DeleteUploadedFiles(logger, hashedFileName)
		if minioErr != nil {
			return minioErr
		}
	}

	err := fileModel.DeleteFileByID(db.Postgresql, fileId)
	if err != nil {
		return err
	}

	if threadId != "" {
		RemoveMediaFileFromThread(db.Elastic, threadId, fileId)
	}

	return nil
}

func RemoveMediaFileFromThread(db *elasticsearch.Client, threadID, fileID string) error {
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

	err := elastic.UpdateDocWithScript(db, models.ThreadIndexName, threadID, req)

	if err != nil {
		return fmt.Errorf("failed to remove media file from thread: %w", err)
	}

	return nil
}
