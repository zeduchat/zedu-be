package models

import (
	"errors"
	"math"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type ChannelExportStatus string

const (
	ExportStatusPending    ChannelExportStatus = "pending"
	ExportStatusInProgress ChannelExportStatus = "in_progress"
	ExportStatusCompleted  ChannelExportStatus = "completed"
	ExportStatusFailed     ChannelExportStatus = "failed"
)

type ChannelExportJobArgs struct {
	ExportID       string `json:"export_id"`
	ChannelID      string `json:"channel_id"`
	UserID         string `json:"user_id"`
	OrganisationID string `json:"organisation_id"`
}

func (ChannelExportJobArgs) Kind() string { return "channel_export" }

type ChannelExport struct {
	ID             string              `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID      string              `gorm:"type:uuid;column:channel_id;not null;index" json:"channel_id"`
	UserID         string              `gorm:"type:uuid;column:user_id;not null;index" json:"user_id"`
	OrganisationID string              `gorm:"type:uuid;column:organisation_id;not null" json:"organisation_id"`
	Status         ChannelExportStatus `gorm:"type:varchar(30);column:status;not null;default:'pending';index" json:"status"`
	FileID         *string             `gorm:"type:uuid;column:file_id" json:"file_id"`
	FileURL        *string             `gorm:"type:text;column:file_url" json:"file_url"`
	ErrorMessage   *string             `gorm:"type:text;column:error_message" json:"error_message,omitempty"`
	CreatedAt      time.Time           `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time           `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	CompletedAt    *time.Time          `gorm:"column:completed_at" json:"completed_at,omitempty"`
}

func (ce *ChannelExport) TableName() string {
	return "channel_exports"
}

func (ce *ChannelExport) BeforeCreate(tx *gorm.DB) error {
	if ce.ID == "" {
		ce.ID = utility.GenerateUUID()
	}
	return nil
}

func (ce *ChannelExport) CreateExport(db *gorm.DB) error {
	return postgresql.CreateOneRecord(db, ce)
}

func (ce *ChannelExport) GetActiveExport(db *gorm.DB, channelID, userID string) (*ChannelExport, error) {
	var export ChannelExport
	err := db.Where("channel_id = ? AND user_id = ? AND status IN (?, ?)", channelID, userID, ExportStatusPending, ExportStatusInProgress).
		Order("created_at DESC").
		First(&export).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &export, nil
}

func (ce *ChannelExport) GetExportByID(db *gorm.DB, exportID string) (*ChannelExport, error) {
	var export ChannelExport
	err := db.Where("id = ?", exportID).First(&export).Error
	if err != nil {
		return nil, err
	}
	return &export, nil
}

func (ce *ChannelExport) GetLatestExport(db *gorm.DB, channelID, userID string) (*ChannelExport, error) {
	var export ChannelExport
	err := db.Where("channel_id = ? AND user_id = ?", channelID, userID).
		Order("created_at DESC").
		First(&export).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &export, nil
}

func (ce *ChannelExport) GetExportHistory(db *gorm.DB, channelID, userID string, page, limit int) ([]ChannelExport, postgresql.PaginationResponse, error) {
	var exports []ChannelExport
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	var totalCount int64
	if err := db.Model(&ChannelExport{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Count(&totalCount).Error; err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	err := db.Where("channel_id = ? AND user_id = ?", channelID, userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&exports).Error
	if err != nil {
		return nil, postgresql.PaginationResponse{}, err
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	pagination := postgresql.PaginationResponse{
		CurrentPage:     page,
		PageCount:       limit,
		TotalPagesCount: totalPages,
	}

	return exports, pagination, nil
}

func (ce *ChannelExport) UpdateStatus(db *gorm.DB, status ChannelExportStatus, fileID *string, fileURL *string, errMsg *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if fileID != nil {
		updates["file_id"] = *fileID
	}
	if fileURL != nil {
		updates["file_url"] = *fileURL
	}
	if errMsg != nil {
		updates["error_message"] = *errMsg
	}
	if status == ExportStatusCompleted || status == ExportStatusFailed {
		now := time.Now()
		updates["completed_at"] = now
	}

	return db.Model(&ChannelExport{}).Where("id = ?", ce.ID).Updates(updates).Error
}
