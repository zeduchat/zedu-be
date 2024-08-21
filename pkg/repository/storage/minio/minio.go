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

func UploadProfilePic(logger *utility.Logger, objectName string, file io.Reader, fileSize int64) (string, error) {

	path1 := "public/profile_pics/" + objectName
	url := ""
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	_, err := minioClient.PutObject(context.Background(), bucketName, path1, file, fileSize, minio.PutObjectOptions{})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", path1, err))
		return url, fmt.Errorf("failed to upload file to %s: %v", path1, err)
	}

	(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", path1))

	url = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, path1)

	return url, nil
}

func DeleteProfilePic(logger *utility.Logger, objectName string) error {
	path := "public/profile_pics/" + objectName
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	err := minioClient.RemoveObject(context.Background(), bucketName, path, minio.RemoveObjectOptions{})

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to delete file %s: %v", path, err))
		return fmt.Errorf("failed to delete file %s: %v", path, err)
	}
	
	return nil
}
