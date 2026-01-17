package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

const (
	avatarBucketName = "telex-avatars"
	avatarPrefix     = "public/avatars/"
)

type AvatarInfo struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified"`
}

func UploadAvatar(logger *utility.Logger, objectName string, file io.Reader, fileSize int64, contentType string) (string, error) {
	path := avatarPrefix + objectName
	minioClient := storage.DB.Minio

	if err := ensureAvatarBucketExists(logger); err != nil {
		return "", err
	}

	_, err := minioClient.StatObject(context.Background(), avatarBucketName, path, minio.StatObjectOptions{})
	if err == nil {
		url := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, avatarBucketName, path)
		utility.LogAndPrint(logger, fmt.Sprintf("Avatar already exists: %s", url))
		return url, nil
	}

	_, err = minioClient.PutObject(context.Background(), avatarBucketName, path, file, fileSize, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to upload avatar to %s: %v", path, err))
		return "", fmt.Errorf("failed to upload avatar to %s: %v", path, err)
	}

	url := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, avatarBucketName, path)
	(*utility.Logger).Info(logger, fmt.Sprintf("Avatar uploaded successfully to %s", path))
	return url, nil
}

func ListAvatars(logger *utility.Logger) ([]AvatarInfo, error) {
	minioClient := storage.DB.Minio

	if err := ensureAvatarBucketExists(logger); err != nil {
		return nil, err
	}

	var avatars []AvatarInfo
	objectCh := minioClient.ListObjects(context.Background(), avatarBucketName, minio.ListObjectsOptions{
		Prefix:    avatarPrefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Error listing avatar: %v", object.Err))
			continue
		}

		if len(object.Key) > 0 && object.Key[len(object.Key)-1] == '/' {
			continue
		}

		name := object.Key
		if len(object.Key) > len(avatarPrefix) {
			name = object.Key[len(avatarPrefix):]
		}

		url := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, avatarBucketName, object.Key)
		avatars = append(avatars, AvatarInfo{
			Name:         name,
			URL:          url,
			Size:         object.Size,
			LastModified: object.LastModified.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	utility.LogAndPrint(logger, fmt.Sprintf("Listed %d avatars", len(avatars)))
	return avatars, nil
}

func AvatarExists(logger *utility.Logger, objectName string) (bool, error) {
	path := avatarPrefix + objectName
	minioClient := storage.DB.Minio

	_, err := minioClient.StatObject(context.Background(), avatarBucketName, path, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		utility.LogAndPrint(logger, fmt.Sprintf("Error checking if avatar %s exists: %v", path, err))
		return false, fmt.Errorf("error checking if avatar %s exists: %v", path, err)
	}
	return true, nil
}

func DeleteAvatar(logger *utility.Logger, objectName string) error {
	path := avatarPrefix + objectName
	minioClient := storage.DB.Minio

	err := minioClient.RemoveObject(context.Background(), avatarBucketName, path, minio.RemoveObjectOptions{})
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to delete avatar %s: %v", path, err))
		return fmt.Errorf("failed to delete avatar %s: %v", path, err)
	}
	return nil
}

func ensureAvatarBucketExists(logger *utility.Logger) error {
	minioClient := storage.DB.Minio

	exists, err := minioClient.BucketExists(context.Background(), avatarBucketName)
	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to check avatar bucket: %v", err))
		return fmt.Errorf("failed to check avatar bucket: %v", err)
	}

	if !exists {
		utility.LogAndPrint(logger, fmt.Sprintf("Creating avatar bucket: %s", avatarBucketName))
		err = minioClient.MakeBucket(context.Background(), avatarBucketName, minio.MakeBucketOptions{})
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("Failed to create avatar bucket: %v", err))
			return fmt.Errorf("failed to create avatar bucket: %v", err)
		}
	}
	return nil
}
