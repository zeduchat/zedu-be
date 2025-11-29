package models

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type UploadRequest struct {
	Files    []*multipart.FileHeader `form:"files" binding:"required"`
	FolderID string                  `form:"folder_id"`
}

type RenameFileRequest struct {
	FileName string `json:"file_name" binding:"required" validate:"required,min=1,max=255"`
}

type File struct {
	ID             string         `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	FileName       string         `gorm:"column:file_name; not null" json:"file_name"`
	FileType       string         `gorm:"column:file_type; type:varchar(50); not null"  json:"file_type"`
	MimeType       string         `gorm:"column:mime_type; type:varchar(50); not null"   json:"mime_type"`
	FileLink       string         `gorm:"column:file_link; type:varchar(200); not null" json:"file_link"`
	Size           int64          `gorm:"column:size" json:"size"`
	OrganisationID string         `gorm:"column:organisation_id; type:uuid; not null" json:"organisation_id"`
	UserID         string         `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	FolderID       *string        `gorm:"column:folder_id; type:uuid" json:"folder_id"`
	ChannelID      *string        `gorm:"column:channel_id; type:uuid" json:"channel_id"`
	MessageID      *string        `gorm:"column:message_id; type:uuid" json:"message_id"`
	CreatedAt      time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; not null; autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type Folder struct {
	ID             string         `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	Name           string         `gorm:"column:name; type:varchar(255); not null" json:"name"`
	OrganisationID string         `gorm:"column:organisation_id; type:uuid; not null" json:"organisation_id"`
	UserID         string         `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	ParentID       *string        `gorm:"column:parent_id; type:uuid" json:"parent_id"`
	CreatedAt      time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; not null; autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type FileType struct {
	MimeType string `json:"mime_type"`
	Category string `json:"category"`
}

func (file *File) CreateFileRecord(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &file)
	if err != nil {
		return err
	}
	return nil
}

func (file *File) GetFileByID(db *gorm.DB, fileID string) (*File, error) {
	query := db.Where("id = ?", fileID)

	err := query.First(&file).Error
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (file *File) GetFileCountByLink(db *gorm.DB, fileLink string) (int64, error) {
	var count int64

	err := db.Model(&file).Where("file_link = ?", fileLink).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (file *File) UpdateFileName(db *gorm.DB, fileID string, newFileName string) error {
	err := db.Model(&File{}).Where("id = ?", fileID).Update("file_name", newFileName).Error
	if err != nil {
		return err
	}
	_, err = file.GetFileByID(db, fileID)
	return err
}

func (file *File) DeleteFileByID(db *gorm.DB, fileID string) error {
	query := db.Where("id = ?", fileID)

	err := query.Delete(&File{}).Error
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
