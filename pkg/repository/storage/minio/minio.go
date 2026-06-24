package minio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

// GetObjectSize retrieves the size of an object in MinIO without downloading it.
func GetObjectSize(bucketName, objectKey string) (int64, error) {
	minioClient := storage.DB.Minio
	if minioClient == nil {
		return 0, fmt.Errorf("minio client not initialized")
	}

	objInfo, err := minioClient.StatObject(context.Background(), bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get object metadata: %w", err)
	}

	return objInfo.Size, nil
}

// GetURLContentLength performs an HTTP HEAD request to fetch the Content-Length of the file.
func GetURLContentLength(fileURL string) (int64, error) {
	resp, err := http.Head(fileURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("http head request failed with status: %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}
