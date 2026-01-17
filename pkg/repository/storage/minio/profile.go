package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

const profilePrefix = "public/profile_pics/"

func UploadProfilePic(logger *utility.Logger, objectName string, file io.Reader, fileSize int64) (string, error) {
	path := profilePrefix + objectName
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	_, err := minioClient.PutObject(context.Background(), bucketName, path, file, fileSize, minio.PutObjectOptions{})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", path, err))
		return "", fmt.Errorf("failed to upload file to %s: %v", path, err)
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", path))

	url := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, path)
	return url, nil
}

func DeleteProfilePic(logger *utility.Logger, objectName string) error {
	path := profilePrefix + objectName
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	err := minioClient.RemoveObject(context.Background(), bucketName, path, minio.RemoveObjectOptions{})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to delete file %s: %v", path, err))
		return fmt.Errorf("failed to delete file %s: %v", path, err)
	}

	return nil
}

func ProfileImageExists(logger *utility.Logger, objectName string) (bool, error) {
	path := profilePrefix + objectName
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	_, err := minioClient.StatObject(context.Background(), bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			utility.LogAndPrint(logger, fmt.Sprintf("Image %s does not exist in bucket %s", path, bucketName))
			return false, nil
		}
		utility.LogAndPrint(logger, fmt.Sprintf("Error checking if image %s exists: %v", path, err))
		return false, fmt.Errorf("error checking if image %s exists: %v", path, err)
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Image %s exists in bucket %s", path, bucketName))
	return true, nil
}
