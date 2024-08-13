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

func UploadProfilePic(objectName string, file io.Reader, fileSize int64) (string, error) {

	path1 := "/public/profile_pics/" + objectName
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
