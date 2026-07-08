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

type FileInfoResponse struct {
	Owner        string    `json:"owner"`
	DateUploaded time.Time `json:"date_uploaded"`
	LastUpdated  time.Time `json:"last_updated"`
	SharedIn     []string  `json:"shared_in"`
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
type FolderWithFilesRequest struct {
	FolderName string                  `form:"folder_name" binding:"required"`
	ParentID   *string                 `form:"parent_id"`
	Files      []*multipart.FileHeader `form:"files" binding:"required"`
}
type UploadFolderWithFilesParams struct {
	FolderName string
	ParentID   *string
	Files      []*multipart.FileHeader
	OrgID      string
	UserID     string
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

type GetFoldersParams struct {
	OrgID       string
	UserID      string
	Page        int
	Limit       int
	QueryParams map[string]string
}

type File struct {
	ID             string         `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	FileName       string         `gorm:"column:file_name; not null" json:"file_name"`
	FileType       string         `gorm:"column:file_type; type:varchar(50); not null"  json:"file_type"`
	MimeType       string         `gorm:"column:mime_type; type:varchar(50); not null"   json:"mime_type"`
	FileLink       string         `gorm:"column:file_link; type:text; not null" json:"file_link"`
	Size           int64          `gorm:"column:size" json:"size"`
	OrganisationID string         `gorm:"column:organisation_id; type:uuid; not null" json:"organisation_id"`
	UserID         string         `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	FolderID       *string        `gorm:"column:folder_id; type:uuid" json:"folder_id"`
	ChannelID      *string        `gorm:"column:channel_id; type:uuid" json:"channel_id"`
	MessageID      *string        `gorm:"column:message_id; type:uuid" json:"message_id"`
	CreatedAt      time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; not null; autoUpdateTime" json:"updated_at"`
	LastAccessedAt *time.Time     `gorm:"column:last_accessed_at" json:"last_accessed_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// New fields for sharing
	AccessType  string `gorm:"column:access_type;type:varchar(20);default:'private';index" json:"access_type"`
	IsShareable bool   `gorm:"column:is_shareable;default:false" json:"is_shareable"`
}

type FileMediaResponse struct {
	ID        string    `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	FileName  string    `gorm:"column:file_name; not null" json:"file_name"`
	FileType  string    `gorm:"column:file_type; type:varchar(50); not null"  json:"file_type"`
	MimeType  string    `gorm:"column:mime_type; type:varchar(50); not null"   json:"mime_type"`
	FileLink  string    `gorm:"column:file_link; type:text; not null" json:"file_link"`
	UserID    string    `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at; not null; autoUpdateTime" json:"updated_at"`
}

type Folder struct {
	ID             string         `gorm:"type:uuid;primary_key" json:"id"`
	OrganisationID string         `gorm:"type:uuid;not null" json:"organisation_id"`
	UserID         string         `gorm:"type:uuid;not null" json:"user_id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	ItemCount      uint64         `json:"item_count" gorm:"-"`
}

type FolderUploadResponse struct {
	Folder    *Folder `json:"folder"`
	Files     []*File `json:"files"`
	FileCount int     `json:"file_count"`
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

func DeleteRecordingFolder(logger *utility.Logger, folderPrefix string) error {
	minioClient := storage.DB.Minio
	bucketName := config.Config.Minio.BucketName
	ctx := context.Background()

	// List all objects (including subdirectory structure) under prefix
	objectsCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    folderPrefix,
		Recursive: true,
	})

	// Batch delete all objects using the channel directly
	errorCh := minioClient.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{})
	
	var lastErr error
	for e := range errorCh {
		if e.Err != nil {
			utility.LogAndPrint(logger, fmt.Sprintf("failed to delete object %s: %v", e.ObjectName, e.Err))
			
			lastErr = e.Err
		}
	}
	
	if lastErr == nil {
		utility.LogAndPrint(logger, fmt.Sprintf("successfully deleted recording folder and all contents: %s", folderPrefix))
	}

	return lastErr
}

// UpdateFilesMetadata updates channel_id and message_id for files attached to messages
// This is called after thread/message creation to associate files with the correct context
func UpdateFilesMetadata(db *gorm.DB, logger *utility.Logger, fileIDs []string, channelID, messageID string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	updates := map[string]interface{}{}
	if channelID != "" {
		updates["channel_id"] = channelID
	}
	if messageID != "" {
		updates["message_id"] = messageID
	}

	if len(updates) == 0 {
		return nil
	}

	err := db.Model(&File{}).
		Where("id IN ?", fileIDs).
		Updates(updates).Error

	if err != nil {
		logger.Error("Failed to update file metadata",
			"file_ids", fileIDs,
			"location", "models.fileManagement.UpdateFilesMetadata",
			"channel_id", channelID,
			"message_id", messageID,
			"error", err)
		return fmt.Errorf("failed to update file metadata: %w", err)
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

// FileShare represents a shared file with specific permissions
type FileShare struct {
	ID             string         `gorm:"type:uuid;primary_key" json:"id"`
	FileID         string         `gorm:"column:file_id;type:uuid;not null;index" json:"file_id"`
	SharedByUserID string         `gorm:"column:shared_by_user_id;type:uuid;not null;index" json:"shared_by_user_id"`
	OrganisationID string         `gorm:"column:organisation_id;type:uuid;not null;index" json:"organisation_id"`
	AccessType     string         `gorm:"column:access_type;type:varchar(20);not null;default:'private';index" json:"access_type"`
	PermissionType string         `gorm:"column:permission_type;type:varchar(20);not null;default:'view'" json:"permission_type"`
	Note           string         `gorm:"column:note;type:text" json:"note"`
	ShareLink      string         `gorm:"column:share_link;type:varchar(255);unique;index" json:"share_link"`
	ExpiresAt      *time.Time     `gorm:"column:expires_at;index" json:"expires_at"`
	AccessCount    int            `gorm:"column:access_count;default:0" json:"access_count"`
	LastAccessedAt *time.Time     `gorm:"column:last_accessed_at" json:"last_accessed_at"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Relationships
	File     *File    `gorm:"foreignKey:FileID" json:"file,omitempty"`
	SharedBy *Profile `gorm:"foreignKey:SharedByUserID" json:"shared_by,omitempty"`
}

func (fs *FileShare) TableName() string {
	return "file_shares"
}

func (fs *FileShare) BeforeCreate(tx *gorm.DB) error {
	if fs.ID == "" {
		fs.ID = utility.GenerateUUID()
	}
	return nil
}

// Update existing File struct to include new fields
type FileWithSharing struct {
	ID             string         `gorm:"column:id; type:uuid; not null; primaryKey; unique;" json:"id"`
	FileName       string         `gorm:"column:file_name; not null" json:"file_name"`
	FileType       string         `gorm:"column:file_type; type:varchar(50); not null"  json:"file_type"`
	MimeType       string         `gorm:"column:mime_type; type:varchar(50); not null"   json:"mime_type"`
	FileLink       string         `gorm:"column:file_link; type:text; not null" json:"file_link"`
	Size           int64          `gorm:"column:size" json:"size"`
	OrganisationID string         `gorm:"column:organisation_id; type:uuid; not null" json:"organisation_id"`
	UserID         string         `gorm:"column:user_id; type:uuid; not null" json:"user_id"`
	FolderID       *string        `gorm:"column:folder_id; type:uuid" json:"folder_id"`
	ChannelID      *string        `gorm:"column:channel_id; type:uuid" json:"channel_id"`
	MessageID      *string        `gorm:"column:message_id; type:uuid" json:"message_id"`
	CreatedAt      time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; not null; autoUpdateTime" json:"updated_at"`
	LastAccessedAt *time.Time     `gorm:"column:last_accessed_at" json:"last_accessed_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// New fields for sharing
	AccessType  string `gorm:"column:access_type;type:varchar(20);default:'private';index" json:"access_type"`
	IsShareable bool   `gorm:"column:is_shareable;default:false" json:"is_shareable"`
}

// Request/Response Models

type ShareFileRequest struct {
	FileID         string     `json:"file_id" validate:"required,uuid"`
	AccessType     string     `json:"access_type" validate:"required,oneof=private public"`
	PermissionType string     `json:"permission_type" validate:"required,oneof=view edit"`
	Note           string     `json:"note" validate:"omitempty,max=500"`
	ExpiresAt      *time.Time `json:"expires_at"`
	ShareViaDM     bool       `json:"share_via_dm"`
	RecipientIDs   []string   `json:"recipient_ids" validate:"omitempty,dive,uuid"`
}

type ShareFileResponse struct {
	FileShareID    string            `json:"file_share_id"`
	FileID         string            `json:"file_id"`
	ShareLink      string            `json:"share_link"`
	AccessType     string            `json:"access_type"`
	PermissionType string            `json:"permission_type"`
	Note           string            `json:"note"`
	ExpiresAt      *time.Time        `json:"expires_at"`
	DMsSent        []string          `json:"dms_sent"`
	RecipientsInfo []DMRecipientInfo `json:"recipients_info,omitempty"`
}

type DMRecipientInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

type UpdateFileShareRequest struct {
	AccessType     *string    `json:"access_type" validate:"omitempty,oneof=private public"`
	PermissionType *string    `json:"permission_type" validate:"omitempty,oneof=view edit"`
	Note           *string    `json:"note" validate:"omitempty,max=500"`
	ExpiresAt      *time.Time `json:"expires_at"`
}

type AccessSharedFileRequest struct {
	ShareLink string `json:"share_link" validate:"required,url"`
}

type FileShareListResponse struct {
	FileShare *FileShare   `json:"file_share"`
	FileInfo  *File        `json:"file_info,omitempty"`
	SharedBy  *ShareByUser `json:"shared_by,omitempty"`
}

type ShareByUser struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type UpdateFileAccessSettingsRequest struct {
	AccessType  *string `json:"access_type" validate:"omitempty,oneof=private public"`
	IsShareable *bool   `json:"is_shareable"`
}

type GetFileSharesRequest struct {
	FileID     string `json:"file_id" validate:"required,uuid"`
	ActiveOnly bool   `json:"active_only"`
}

type FileSharedEvent struct {
	FileID     string `json:"file_id"`
	SharedBy   string `json:"shared_by"`
	Event      string `json:"event"`
	Permission string `json:"permission"`
	AccessType string `json:"access_type"`
}

// Helper functions for validation

func ValidateAccessType(accessType string) error {
	validTypes := map[string]bool{
		"private": true,
		"public":  true,
	}
	if !validTypes[accessType] {
		return fmt.Errorf("invalid access_type: %s, must be 'private' or 'public'", accessType)
	}
	return nil
}

func ValidatePermissionType(permissionType string) error {
	validTypes := map[string]bool{
		"view": true,
		"edit": true,
	}
	if !validTypes[permissionType] {
		return fmt.Errorf("invalid permission_type: %s, must be 'view' or 'edit'", permissionType)
	}
	return nil
}

func ValidateShareExpiration(expiresAt *time.Time) error {
	if expiresAt == nil {
		return nil
	}

	now := time.Now().UTC()
	if expiresAt.Before(now) {
		return fmt.Errorf("expiration date cannot be in the past")
	}

	maxFuture := now.AddDate(1, 0, 0)
	if expiresAt.After(maxFuture) {
		return fmt.Errorf("expiration date cannot be more than 1 year in the future")
	}

	return nil
}

// MatchesMediaType checks if a mimeType matches the requested mediaType
func MatchesMediaType(mimeType, mediaType string) bool {
	switch mediaType {
	case "images":
		return len(mimeType) >= 6 && mimeType[:6] == "image/"
	case "videos":
		return len(mimeType) >= 6 && mimeType[:6] == "video/"
	case "audio":
		return len(mimeType) >= 6 && mimeType[:6] == "audio/"
	case "documents":
		return ContainsAny(mimeType, []string{"pdf", "document", "word", "text", "rtf"})
	default:
		return true
	}
}

func ContainsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
