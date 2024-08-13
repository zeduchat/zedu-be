package minio

import (
	"context"
	"fmt"
	"log"

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
		Secure: true,
	})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to initialize MinIO client: %v", err))
		return nil
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Successfully connected to MinIO at %s", vsn.MinioEndpoint))

	exists, err := minioClient.BucketExists(context.Background(), vsn.BucketName)
	if err != nil {
		log.Fatalf("Failed to check if bucket exists: %v", err)
		return nil
	}

	if exists {
		utility.LogAndPrint(logger, fmt.Sprintf("Bucket %s exists", vsn.BucketName))
		return nil
	} else {
		utility.LogAndPrint(logger, fmt.Sprintf("Bucket %s does not exist, creating it...", vsn.BucketName))

		err = minioClient.MakeBucket(context.Background(), vsn.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Failed to create bucket %s: %v", vsn.BucketName, err))
		}

		utility.LogAndPrint(logger, fmt.Sprintf("Successfully created bucket %s", vsn.BucketName))
	}

	storage.DB.Minio = minioClient

	return minioClient
}