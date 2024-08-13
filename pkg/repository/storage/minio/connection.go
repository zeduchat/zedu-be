package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func ConnectToMinio(logger *utility.Logger, configBucket config.Minio) *minio.Client {
	vsn := configBucket
	minioClient, err := minio.New(vsn.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(vsn.AccessKey, vsn.Secret, ""),
		Secure: false,
	})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to initialize MinIO client: %v", err))
		return nil
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Successfully connected to MinIO at %s", vsn.MinioEndpoint))

	exists, err := minioClient.BucketExists(context.Background(), vsn.BucketName)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to check if bucket exists: %v", err))
		return nil
	}

	if exists {
		utility.LogAndPrint(logger, fmt.Sprintf("Bucket %s exists", vsn.BucketName))
		return nil
	}

	storage.DB.Minio = minioClient
	storage.Logger = logger

	return minioClient
}

func UploadProfilePic(objectName string, file io.Reader, fileSize int64) (string, error) {

	path1 := "profile_pics/" + objectName
	url := ""
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	_, err := minioClient.PutObject(context.Background(), bucketName, path1, file, fileSize, minio.PutObjectOptions{})
	if err != nil {
		utility.LogAndPrint(storage.Logger, fmt.Sprintf("failed to upload file to %s: %v", path1, err))
		return url, fmt.Errorf("failed to upload file to %s: %v", path1, err)
	}
	utility.LogAndPrint(storage.Logger, fmt.Sprintf("File uploaded successfully to %s\n", path1))
	url = fmt.Sprintf("http://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, path1)
	return url, nil
}
