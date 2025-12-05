package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type PinnedFile struct {
	ID             string    `gorm:"type:uuid;primary_key" json:"id"`
	FileID         string    `gorm:"type:uuid;not null;index" json:"file_id"`
	UserID         string    `gorm:"type:uuid;not null;index" json:"user_id"`
	OrganisationID string    `gorm:"type:uuid;not null;index" json:"organisation_id"`
	PinnedAt       time.Time `gorm:"autoCreateTime" json:"pinned_at"`
}

func (p *PinnedFile) PinFile(db *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}

	if p.CheckPinnedFileExists(db) {
		return nil // idempotent, already pinned
	}

	return postgresql.CreateOneRecord(db, p)
}

func (p *PinnedFile) UnpinFile(db *gorm.DB) error {
	return db.Where("organisation_id = ? AND user_id = ? AND file_id = ?", p.OrganisationID, p.UserID, p.FileID).Delete(&PinnedFile{}).Error
}

func (p *PinnedFile) GetPinnedFiles(db *gorm.DB, userID, orgID string) ([]File, error) {
	var files []File
	err := db.Table("files").
		Joins("JOIN pinned_files ON pinned_files.file_id = files.id").
		Where("pinned_files.organisation_id = ? AND pinned_files.user_id = ?", orgID, userID).
		Find(&files).Error

	if err != nil {
		return nil, err
	}
	return files, nil
}

func (p *PinnedFile) CheckPinnedFileExists(db *gorm.DB) bool {
	var existing PinnedFile
	exists := postgresql.CheckExists(db, &existing, "organisation_id = ? AND user_id = ? AND file_id = ?", p.OrganisationID, p.UserID, p.FileID)
	if exists {
		*p = existing
	}

	return exists
}
