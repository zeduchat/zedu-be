package minio

import (
	// "context"
	"context"
	"fmt"

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
		// Secure: true,
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
	} else {
		utility.LogAndPrint(logger, fmt.Sprintf("Bucket does not %s exists", vsn.BucketName))
		return nil
	}

	storage.DB.Minio = minioClient

	return minioClient
}
