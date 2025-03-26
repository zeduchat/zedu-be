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

var AllowedFileTypes = map[string]string{
	// Images
	"image/png":  "public/file-uploads/",
	"image/jpeg": "public/file-uploads/",
	"image/jpg":  "public/file-uploads/",
	"image/gif":  "public/file-uploads/",
	"image/webp": "public/file-uploads/",

	// Documents
	"text/plain":         "public/file-uploads/", // .txt or .csv
	"text/csv":           "public/file-uploads/",
	"application/pdf":    "public/file-uploads/",
	"application/msword": "public/file-uploads/",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "public/file-uploads/", // .docx
	"application/vnd.ms-word.document.macroEnabled.12":                        "public/file-uploads/", // .docm
	"application/x-msword":     "public/file-uploads/", // Alternative .doc MIME
	"application/zip":          "public/file-uploads/", // Some .docx files are detected as ZIP
	"application/vnd.ms-excel": "public/file-uploads/", // .xls
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "public/file-uploads/", // .xlsx
	"application/vnd.ms-powerpoint":                                             "public/file-uploads/", // .ppt
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "public/file-uploads/", // .pptx

	// Audio
	"audio/mpeg": "public/file-uploads/", // .mp3
	"audio/wav":  "public/file-uploads/", // .wav

	// Video
	"video/mp4":  "public/file-uploads/", // .mp4
	"video/webm": "public/file-uploads/", // .webm
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
