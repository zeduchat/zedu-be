package models

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"
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

type UploadFileParams struct {
	File     multipart.File
	Header   *multipart.FileHeader
	FolderID string
	OrgID    string
	UserID   string
}

type CreateFolderParams struct {
	Name           string
	Description    string
	OrganisationID string
	UserID         string
}

type DeleteMultipleFilesRequest struct {
	IDs       []string `json:"ids" validate:"required,min=1,dive,uuid"`
	Permanent bool     `json:"permanent"`
}

type DeleteMultipleFoldersRequest struct {
	IDs       []string `json:"ids" validate:"required,min=1,dive,uuid"`
	Permanent bool     `json:"permanent"`
}

type UpdateFileNameParams struct {
	FileID      string
	NewFileName string
	OrgID       string
	UserID      string
	FolderID    string
}

type RenameFolderRequest struct {
	FolderName string `json:"folder_name" binding:"required" validate:"required,min=1,max=255"`
}

type UpdateFolderParams struct {
	FolderID string
	Name     string
	OrgID    string
	UserID   string
}

type GetFilesParams struct {
	OrgID       string
	UserID      string
	QueryParams map[string]string
	Page        int
	Limit       int
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
	ID             string         `gorm:"type:uuid;primary_key" json:"id"`
	OrganisationID string         `gorm:"type:uuid;not null" json:"organisation_id"`
	UserID         string         `gorm:"type:uuid;not null" json:"user_id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	Description    string         `gorm:"type:text" json:"description"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	ItemCount      uint64         `json:"item_count" gorm:"-"`
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
	file.FileName = newFileName
	return nil
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

func (file *File) GetFileCategory() string {
	mimeType := strings.ToLower(file.MimeType)

	if strings.Contains(mimeType, "pdf") ||
		strings.Contains(mimeType, "document") ||
		strings.Contains(mimeType, "word") ||
		strings.Contains(mimeType, "text") ||
		strings.Contains(mimeType, "rtf") ||
		strings.HasSuffix(mimeType, "doc") ||
		strings.HasSuffix(mimeType, "docx") ||
		strings.HasSuffix(mimeType, "txt") ||
		strings.HasSuffix(mimeType, "odt") {
		return "documents"
	}

	if strings.Contains(mimeType, "spreadsheet") ||
		strings.Contains(mimeType, "excel") ||
		strings.HasSuffix(mimeType, "xls") ||
		strings.HasSuffix(mimeType, "xlsx") ||
		strings.HasSuffix(mimeType, "csv") ||
		strings.HasSuffix(mimeType, "ods") {
		return "spreadsheets"
	}

	if strings.HasPrefix(mimeType, "image/") {
		return "images"
	}

	if strings.HasPrefix(mimeType, "video/") {
		return "videos"
	}

	if strings.HasPrefix(mimeType, "audio/") {
		return "music"
	}

	return "other"
}

func GetDateRangeFilter(filter string) (start, end *time.Time) {
	now := time.Now()

	switch strings.ToLower(filter) {
	case "today":
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)
		return &startOfDay, &endOfDay

	case "last_7_days":
		start := now.AddDate(0, 0, -7)
		return &start, &now

	case "last_30_days":
		start := now.AddDate(0, 0, -30)
		return &start, &now

	case "this_year":
		startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return &startOfYear, &now

	case "last_year":
		startOfLastYear := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
		endOfLastYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return &startOfLastYear, &endOfLastYear
	}

	return nil, nil
}
