package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type MediaPreferences struct {
	ID                    string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	UserID                uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceID              *uuid.UUID     `gorm:"type:uuid;null;index" json:"device_id"`
	AutoDownloadPhotos    string         `gorm:"type:varchar(20);default:'wifi_only'" json:"auto_download_photos"`
	AutoDownloadAudio     string         `gorm:"type:varchar(20);default:'wifi_only'" json:"auto_download_audio"`
	AutoDownloadDocuments string         `gorm:"type:varchar(20);default:'wifi_only'" json:"auto_download_documents"`
	AutoDownloadVideos    string         `gorm:"type:varchar(20);default:'wifi_only'" json:"auto_download_videos"`
	UploadQuality         string         `gorm:"type:varchar(20);default:'standard'" json:"upload_quality"`
	CreatedAt             time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for GORM
func (MediaPreferences) TableName() string {
	return "media_preferences"
}

// UpdateMediaPreferencesRequest represents the request body for updating media preferences
type UpdateMediaPreferencesRequest struct {
	DeviceID              *string `json:"device_id" validate:"omitempty,uuid"`
	AutoDownloadPhotos    string  `json:"auto_download_photos" validate:"omitempty,oneof=always wifi_only never"`
	AutoDownloadAudio     string  `json:"auto_download_audio" validate:"omitempty,oneof=always wifi_only never"`
	AutoDownloadDocuments string  `json:"auto_download_documents" validate:"omitempty,oneof=always wifi_only never"`
	AutoDownloadVideos    string  `json:"auto_download_videos" validate:"omitempty,oneof=always wifi_only never"`
	UploadQuality         string  `json:"upload_quality" validate:"omitempty,oneof=standard high original"`
}

// MediaPreferencesResponse represents the response for media preferences
type MediaPreferencesResponse struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	DeviceID              *string   `json:"device_id"`
	AutoDownloadPhotos    string    `json:"auto_download_photos"`
	AutoDownloadAudio     string    `json:"auto_download_audio"`
	AutoDownloadDocuments string    `json:"auto_download_documents"`
	AutoDownloadVideos    string    `json:"auto_download_videos"`
	UploadQuality         string    `json:"upload_quality"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// GetMediaPreferencesRequest represents the request for getting media preferences
type GetMediaPreferencesRequest struct {
	DeviceID *string `json:"device_id" form:"device_id" validate:"omitempty,uuid"`
}

// ResetAutoDownloadRequest represents the request body for resetting auto-download settings
type ResetAutoDownloadRequest struct {
	DeviceID *string `json:"device_id" validate:"omitempty,uuid"`
}

// Create creates a new media preferences record
func (m *MediaPreferences) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, m)
	if err != nil {
		return err
	}
	return nil
}

// Update updates an existing media preferences record (partial update)
func (m *MediaPreferences) Update(db *gorm.DB, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	err := db.Model(m).Updates(updates).Error
	if err != nil {
		return err
	}
	return nil
}

// GetUserPreferences retrieves user-level media preferences
func (m *MediaPreferences) GetUserPreferences(db *gorm.DB, userID string) (MediaPreferences, error) {
	var pref MediaPreferences

	err := db.Where("user_id = ? AND device_id IS NULL", userID).First(&pref).Error
	if err != nil {
		return pref, err
	}

	return pref, nil
}

// GetDevicePreferences retrieves device-specific preferences with fallback to user-level
func (m *MediaPreferences) GetDevicePreferences(db *gorm.DB, userID string, deviceID *string) (MediaPreferences, error) {
	var devicePref, userPref MediaPreferences
	var result MediaPreferences

	// First, try to get user-level preferences as fallback
	err := db.Where("user_id = ? AND device_id IS NULL", userID).First(&userPref).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return result, err
	}

	// If device_id is provided, try to get device-specific preferences
	if deviceID != nil {
		deviceUUID, err := uuid.FromString(*deviceID)
		if err != nil {
			return result, fmt.Errorf("invalid device_id format")
		}

		deviceErr := db.Where("user_id = ? AND device_id = ?", userID, deviceUUID).First(&devicePref).Error
		if deviceErr != nil && !errors.Is(deviceErr, gorm.ErrRecordNotFound) {
			return result, deviceErr
		}

		// If device preferences found, merge with user-level (device takes precedence)
		if !errors.Is(deviceErr, gorm.ErrRecordNotFound) {
			result = devicePref
			// Fill in missing fields from user-level preferences
			if result.AutoDownloadPhotos == "" && userPref.AutoDownloadPhotos != "" {
				result.AutoDownloadPhotos = userPref.AutoDownloadPhotos
			}
			if result.AutoDownloadAudio == "" && userPref.AutoDownloadAudio != "" {
				result.AutoDownloadAudio = userPref.AutoDownloadAudio
			}
			if result.AutoDownloadDocuments == "" && userPref.AutoDownloadDocuments != "" {
				result.AutoDownloadDocuments = userPref.AutoDownloadDocuments
			}
			if result.AutoDownloadVideos == "" && userPref.AutoDownloadVideos != "" {
				result.AutoDownloadVideos = userPref.AutoDownloadVideos
			}
			if result.UploadQuality == "" && userPref.UploadQuality != "" {
				result.UploadQuality = userPref.UploadQuality
			}
			return result, nil
		}
	}

	// If no device preferences found (or device_id not provided), return user-level preferences if they exist
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return userPref, nil
	}

	// If no preferences exist at all, return empty result
	return result, gorm.ErrRecordNotFound
}

// GetOrCreateUserPreferences gets user-level preferences or creates default if not exists
func (m *MediaPreferences) GetOrCreateUserPreferences(db *gorm.DB, userID string) (MediaPreferences, error) {
	var pref MediaPreferences

	err := db.Where("user_id = ? AND device_id IS NULL", userID).First(&pref).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create default user preferences
		userUUID, err := uuid.FromString(userID)
		if err != nil {
			return pref, fmt.Errorf("invalid user_id format")
		}

		pref = MediaPreferences{
			ID:                    utility.GenerateUUID(),
			UserID:                userUUID,
			DeviceID:              nil,
			AutoDownloadPhotos:    "wifi_only",
			AutoDownloadAudio:     "wifi_only",
			AutoDownloadDocuments: "wifi_only",
			AutoDownloadVideos:    "wifi_only",
			UploadQuality:         "standard",
		}

		if err := pref.Create(db); err != nil {
			return pref, err
		}

		return pref, nil
	}

	if err != nil {
		return pref, err
	}

	return pref, nil
}

// ValidateUploadQuality validates the upload quality value
func ValidateUploadQuality(quality string) error {
	validValues := []string{"standard", "high", "original"}
	for _, v := range validValues {
		if quality == v {
			return nil
		}
	}
	return fmt.Errorf("invalid upload quality. Must be one of: standard, high, original")
}

// ValidateAutoDownload validates the auto-download value
func ValidateAutoDownload(value string) error {
	validValues := []string{"always", "wifi_only", "never"}
	for _, v := range validValues {
		if value == v {
			return nil
		}
	}
	return fmt.Errorf("invalid auto-download value. Must be one of: always, wifi_only, never")
}

// ToResponse converts MediaPreferences to MediaPreferencesResponse
func (m *MediaPreferences) ToResponse() MediaPreferencesResponse {
	var deviceIDStr *string
	if m.DeviceID != nil {
		deviceID := m.DeviceID.String()
		deviceIDStr = &deviceID
	}

	return MediaPreferencesResponse{
		ID:                    m.ID,
		UserID:                m.UserID.String(),
		DeviceID:              deviceIDStr,
		AutoDownloadPhotos:    m.AutoDownloadPhotos,
		AutoDownloadAudio:     m.AutoDownloadAudio,
		AutoDownloadDocuments: m.AutoDownloadDocuments,
		AutoDownloadVideos:    m.AutoDownloadVideos,
		UploadQuality:         m.UploadQuality,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}
