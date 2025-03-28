package models

import (
	"mime/multipart"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files" binding:"required"`
}

type UploadedFileResponse struct {
	ID       string `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	FileName string `gorm:"column:file_name; unique; not null" json:"file_name"`
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

func (file *UploadedFileResponse) DeleteFileByID(db *gorm.DB, fileID string) error {
	query := db.Where("id = ?", fileID)

	err := query.Delete(&UploadedFileResponse{}).Error
	if err != nil {
		return err
	}

	return nil
}
