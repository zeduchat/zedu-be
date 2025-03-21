package fileManagement

import (
	"context"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
)

func FormatFileName(filename string) string {
	return strings.ReplaceAll(filename, " ", "_")
}

func GetFileCategory(mimeType string) (string, bool) {
	for _, fileType := range models.AllowedFileTypes {
		if fileType.MimeType == mimeType {
			return fileType.Category, true
		}
	}
	return "", false
}

const maxFileSize = 100 * 1024 * 1024

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
		return "application/octet-stream", nil // Fallback
	}

	return mimeType, nil
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

func UploadFiles(logger *utility.Logger, file multipart.File, header *multipart.FileHeader) (string, error) {
	var generatedUrl string
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	if header.Size > maxFileSize {
		utility.LogAndPrint(logger, fmt.Sprintf("file exceeds max size"))
		return "", fmt.Errorf("file exceeds max size")
	}

	mimeType, mimeTypeErr := DetectMimeType(file)
	if mimeTypeErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("could not detect file type: %v", mimeTypeErr))
		return "", fmt.Errorf("could not detect file type: %v", mimeTypeErr)
	}

	storagePath, valid := GetFileCategory(mimeType)
	if !valid {
		utility.LogAndPrint(logger, fmt.Sprintf("invalid file type"))
		return "", fmt.Errorf("invalid file type")
	}
	if storagePath == "" {
		utility.LogAndPrint(logger, fmt.Sprintf("Could not find storage path for file type"))
		return "", fmt.Errorf("could not find storage path for file type")
	}

	fullPath := storagePath + header.Filename
	encodedFilePath := FormatFileName(fullPath)

	exists, existsErr := FileExists(logger, encodedFilePath)
	if existsErr != nil && exists {
		utility.LogAndPrint(logger, fmt.Sprintf("file existence error: %v", existsErr))
		return "", existsErr
	}

	_, err := minioClient.PutObject(context.Background(), bucketName, encodedFilePath, file, header.Size, minio.PutObjectOptions{ContentType: mimeType})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", encodedFilePath, err))
		return "", fmt.Errorf("failed to upload file to %s: %v", encodedFilePath, err)
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", encodedFilePath))

	generatedUrl = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

	return generatedUrl, nil
}

func GeneratePresignedURL(logger *utility.Logger, objectName string) (string, error) {
	mimeType, mimeTypeErr := GetMimeTypeFromFileName(objectName)
	if mimeTypeErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Could not detect file type: %v", mimeTypeErr))
		return "", fmt.Errorf("could not detect file type: %v", mimeTypeErr)
	}

	storagePath, valid := GetFileCategory(mimeType)
	if !valid {
		utility.LogAndPrint(logger, fmt.Sprintf("Invalid file type"))
		return "", fmt.Errorf("invalid file type")
	}
	if storagePath == "" {
		utility.LogAndPrint(logger, fmt.Sprintf("Could not find storage path for file type"))
		return "", fmt.Errorf("could not find storage path for file type")
	}

	fullPath := storagePath + objectName
	encodedFilePath := FormatFileName(fullPath)

	exists, existsErr := FileExists(logger, fullPath)
	if existsErr != nil && exists {
		// Set expiration time (e.g. 30 minutes)
		expiry := 30 * time.Minute
		minioClient := storage.DB.Minio
		bucketName := config.Config.Minio.BucketName

		presignedURL, err := minioClient.PresignedGetObject(context.Background(), bucketName, encodedFilePath, expiry, nil)
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Error retrieving URL: %v", err))
			return "", err
		}
		return presignedURL.String(), nil
	} else {
		utility.LogAndPrint(logger, fmt.Sprintf("File does not exist"))
		return "", fmt.Errorf("file does not exist")
	}

}
