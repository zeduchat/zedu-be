package fileManagement

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
)

func GetFileCategory(mimeType string) (string, bool) {
	for _, fileType := range models.AllowedFileTypes {
		if fileType.MimeType == mimeType {
			return fileType.Category, true
		}
	}
	return "", false
}

const maxFileSize = 50 * 1024 * 1024

// Scan file with ClamAV before uploading
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
	var url string
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	if header.Size > maxFileSize {
		return "", fmt.Errorf("file exists max size")
	}

	// scanErr := minioService.ScanFileWithClamAV(file)
	// if scanErr != nil {
	// 	return "", scanErr
	// }

	mimeType, mimeTypeErr := DetectMimeType(file)
	if mimeTypeErr != nil {
		return "", fmt.Errorf("could not detect file type: %v", mimeTypeErr)
	}

	_, existsErr := FileExists(logger, header.Filename)
	if existsErr != nil {
		return "", existsErr
	}

	storagePath, valid := GetFileCategory(mimeType)
	if !valid {
		return "", fmt.Errorf("invalid file type")
	}

	_, err := minioClient.PutObject(context.Background(), bucketName, storagePath, file, header.Size, minio.PutObjectOptions{ContentType: mimeType})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", storagePath, err))
		return "", fmt.Errorf("failed to upload file to %s: %v", storagePath, err)
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", storagePath))

	url = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, storagePath)

	return url, nil
}

func GeneratePresignedURL(logger *utility.Logger, objectName string) (string, error) {
	exists, err := FileExists(logger, objectName)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("file %s does not exist in the bucket", objectName)
	}

	// Set expiration time (e.g. 30 minutes)
	expiry := 30 * time.Minute
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	presignedURL, err := minioClient.PresignedGetObject(context.Background(), bucketName, objectName, expiry, nil)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}
