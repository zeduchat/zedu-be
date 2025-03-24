package fileManagement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

func FormatFileName(filename string) string {
	return strings.ReplaceAll(filename, " ", "_")
}

func GetFileCategory(mimeType string) (string, bool) {
	if strings.Contains(mimeType, ";") {
		mimeType = strings.Split(mimeType, ";")[0]
	}
	category, exists := models.AllowedFileTypes[mimeType]
	return category, exists
}

const maxFileSize = 100 * 1024 * 1024

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

func GetMimeTypeFromFileName(filename string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "", fmt.Errorf("filename has no extension")
	}

	mimeType := mime.TypeByExtension(strings.ToLower(ext))
	if mimeType == "" {
		return "application/octet-stream", nil // Fallback
	}

	return mimeType, nil
}

func HashFile(file multipart.File) (string, error) {
	hasher := sha256.New()

	// Copy file content into the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	// Convert hash to a hexadecimal string
	return hex.EncodeToString(hasher.Sum(nil)), nil
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

func UploadFiles(db *gorm.DB, logger *utility.Logger, file multipart.File, header *multipart.FileHeader) (*models.UploadedFileResponse, error) {
	var generatedUrl string
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName

	if header.Size > maxFileSize {
		errMsg := "file exceeds max size"
		utility.LogAndPrint(logger, errMsg)
		return nil, fmt.Errorf("file exceeds max size")
	}

	// Compute SHA256 hash of the file
	fileHash, hashErr := HashFile(file)
	if hashErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("failed to hash file: %v", hashErr))
		return nil, fmt.Errorf("failed to hash file: %v", hashErr)
	}

	// Reset file pointer to the beginning after reading the file to the end
	if seeker, ok := file.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("failed to seek file: %v", err))
			return nil, fmt.Errorf("failed to seek file: %v", err)
		}
	}

	mimeType, mimeTypeErr := DetectMimeType(file)
	if mimeTypeErr != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("could not detect file type: %v", mimeTypeErr.Error()))
		return nil, fmt.Errorf("could not detect file type: %w", mimeTypeErr)
	}

	storagePath, valid := GetFileCategory(mimeType)
	if !valid {
		utility.LogAndPrint(logger, "invalid file type")
		return nil, fmt.Errorf("invalid file type")
	}
	if storagePath == "" {
		utility.LogAndPrint(logger, "Could not find storage path for file type")
		return nil, fmt.Errorf("could not find storage path for file type")
	}

	extension := filepath.Ext(header.Filename)

	hashedFileName := fmt.Sprintf("%s%s", fileHash, extension)
	encodedFilePath := storagePath + hashedFileName

	exists, existsErr := FileExists(logger, encodedFilePath)
	if existsErr != nil && !exists {
		utility.LogAndPrint(logger, fmt.Sprintf("error: %v. using existing file reference", existsErr.Error()))
		return nil, existsErr
	} else if exists {
		utility.LogAndPrint(logger, "using existing file reference")
		existingFileURL := fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

		return nil, fmt.Errorf("file already exists at %v", existingFileURL)
	} else {
		_, err := minioClient.PutObject(context.Background(), bucketName, encodedFilePath, file, header.Size, minio.PutObjectOptions{ContentType: mimeType})
		if err != nil {
			errMsg := fmt.Errorf("failed to upload file to %s: %w", encodedFilePath, err)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to upload file to %s: %v", encodedFilePath, err.Error()))
			return nil, errMsg
		}

		(*utility.Logger).Info(logger, fmt.Sprintf("File uploaded successfully to %s\n", encodedFilePath))

		generatedUrl = fmt.Sprintf("https://%s/%s/%s", minioClient.EndpointURL().Host, bucketName, encodedFilePath)

		response := models.UploadedFileResponse{
			FileName: header.Filename,
			FileType: filepath.Ext(header.Filename)[1:],
			MimeType: mimeType,
			FileLink: generatedUrl,
		}

		storageErr := response.CreateFileRecord(db)
		if storageErr != nil {
			errMsg := fmt.Errorf("error saving file details: %w", storageErr)
			utility.LogAndPrint(logger, fmt.Sprintf("failed to save file details to database: %v", errMsg.Error()))
			return nil, errMsg
		}

		return &response, nil
	}
}
