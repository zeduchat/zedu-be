package models

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/hngprojects/telex_be/internal/config"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files" binding:"required"`
}

type RenameFileRequest struct {
	FileName string `json:"file_name" binding:"required" validate:"required,min=1,max=255"`
}

type UploadedFileResponse struct {
	ID       string `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	FileName string `gorm:"column:file_name; not null" json:"file_name"`
	FileType string `gorm:"column:file_type; type:varchar(50); not null"  json:"file_type"`
	MimeType string `gorm:"column:mime_type; type:varchar(50); not null"   json:"mime_type"`
	FileLink string `gorm:"column:file_link; type:varchar(200); not null" json:"file_link"`
}

type FileType struct {
	MimeType string `json:"mime_type"`
	Category string `json:"category"`
}

func (file *UploadedFileResponse) CreateFileRecord(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &file)
	if err != nil {
		return err
	}
	return nil
}

func (file *UploadedFileResponse) GetFileByID(db *gorm.DB, fileID string) (*UploadedFileResponse, error) {
	query := db.Where("id = ?", fileID)

	err := query.First(&file).Error
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (file *UploadedFileResponse) GetFileCountByLink(db *gorm.DB, fileLink string) (int64, error) {
	var count int64

	err := db.Model(&file).Where("file_link = ?", fileLink).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (file *UploadedFileResponse) UpdateFileName(db *gorm.DB, fileID string, newFileName string) error {
	err := db.Model(&UploadedFileResponse{}).Where("id = ?", fileID).Update("file_name", newFileName).Error
	if err != nil {
		return err
	}
	_, err = file.GetFileByID(db, fileID)
	return err
}

func (file *UploadedFileResponse) DeleteFileByID(db *gorm.DB, fileID string) error {
	query := db.Where("id = ?", fileID)

	err := query.Delete(&UploadedFileResponse{}).Error
	if err != nil {
		return err
	}

	return nil
}

func DeleteUploadedFiles(logger *utility.Logger, fileName string) error {
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName
	encodedFilePath := "public/file-uploads/" + fileName

	err := minioClient.RemoveObject(context.Background(), bucketName, encodedFilePath, minio.RemoveObjectOptions{})
	if err != nil {
		errMsg := fmt.Errorf("failed to delete file: %w", err)
		utility.LogAndPrint(logger, fmt.Sprintf("failed to delete file: %v", err.Error()))
		return errMsg
	}

	return nil
}
