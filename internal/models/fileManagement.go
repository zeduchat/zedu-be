package models

import (
	"mime/multipart"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files" binding:"required"` // Multiple file headers
}

type UploadedFileResponse struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	FileName string `gorm:"uniqueIndex" json:"file_name"`
	FileType string `json:"file_type"`
	MimeType string `json:"mime_type"`
	FileLink string `json:"file_link"`
}

type FileType struct {
	MimeType string `json:"mime_type"`
	Category string `json:"category"`
}

var AllowedFileTypes = map[string]string{
	// Images
	"image/png":  "file-uploads/",
	"image/jpeg": "file-uploads/",
	"image/jpg":  "file-uploads/",
	"image/gif":  "file-uploads/",
	"image/webp": "file-uploads/",

	// Documents
	"text/plain":         "file-uploads/", // .txt or .csv
	"text/csv":           "file-uploads/",
	"application/pdf":    "file-uploads/",
	"application/msword": "file-uploads/",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "file-uploads/", // .docx
	"application/vnd.ms-word.document.macroEnabled.12":                        "file-uploads/", // .docm
	"application/x-msword":     "file-uploads/", // Alternative .doc MIME
	"application/zip":          "file-uploads/", // Some .docx files are detected as ZIP
	"application/vnd.ms-excel": "file-uploads/", // .xls
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "file-uploads/", // .xlsx
	"application/vnd.ms-powerpoint":                                             "file-uploads/", // .ppt
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "file-uploads/", // .pptx

	// Audio
	"audio/mpeg": "file-uploads/", // .mp3
	"audio/wav":  "file-uploads/", // .wav

	// Video
	"video/mp4":  "file-uploads/", // .mp4
	"video/webm": "file-uploads/", // .webm
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
