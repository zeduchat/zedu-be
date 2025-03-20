package fileManagement

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/dutchcoders/go-clamd"
	"github.com/hngprojects/telex_be/internal/config"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	minioService "github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
)


var AllowedMimeTypes = map[string]string{
	// Image
	"image/png":  "images",
	"image/jpeg": "images",
	"image/jpg":  "images",

	// Document
	"text/csv":        "documents",
	"application/pdf": "documents",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "documents",

	// Audio
	"audio/mpeg": "audio",
	"audio/wav":  "audio",

	// Video
	"video/mp4":       "videos",
	"video/quicktime": "videos",
}

const maxFileSize = 50 * 1024 * 1024

// Scan file with ClamAV before uploading
func ScanFileWithClamAV(file multipart.File) error {
	if minioService.ClamAV == nil {
		return fmt.Errorf("ClamAV is not initialized")
	}

	response, err := minioService.ClamAV.ScanStream(file, make(chan bool)) // Use global clamAV instance
	if err != nil {
		return fmt.Errorf("ClamAV scan failed: %v", err)
	}

	for result := range response {
		if result.Status == clamd.RES_FOUND || result.Status == "FOUND" {
			return fmt.Errorf("malware detected: %s", result.Description)
		}
	}

	return nil
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

func FileExists(fileName string) (bool, error) {
	var logger *utility.Logger
	path := "public/uploads/" + fileName
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	if minioClient == nil || bucketName == "" {
		return false, fmt.Errorf("minio is not properly initialized")
	}

	_, err := minioClient.StatObject(context.Background(), bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			utility.LogAndPrint(logger, fmt.Sprintf("File %s does not exist in bucket %s", path, bucketName))
			return false, nil
		}
		utility.LogAndPrint(logger, fmt.Sprintf("Error checking if file %s exists: %v", path, err))
		return false, fmt.Errorf("error checking if file %s exists: %v", path, err)
	}

	utility.LogAndPrint(logger, fmt.Sprintf("File %s exists in bucket %s", path, bucketName))
	return true, fmt.Errorf("file %s exists in bucket %s", path, bucketName)
}

func UploadFiles(file multipart.File, header *multipart.FileHeader) (string, error) {
	var logger *utility.Logger
	var url string

	if header.Size > maxFileSize {
		return "", fmt.Errorf("file exists max size")
	}
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	if minioClient == nil || bucketName == "" {
		return "", fmt.Errorf("minio is not properly initialized")
	}

	scanErr := ScanFileWithClamAV(file)
	if scanErr != nil {
		return "", scanErr
	}

	mimeType, mimeTypeErr := DetectMimeType(file)
	if mimeTypeErr != nil {
		return "", fmt.Errorf("could not detect file type: %v", mimeTypeErr)
	}

	_, existsErr := FileExists(header.Filename)
	if existsErr != nil {
		return "", existsErr
	}

	storagePath, valid := AllowedMimeTypes[mimeType]
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

func GeneratePresignedURL(objectName string) (string, error) {
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
