package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type DeviceRegistry struct {
	ID          string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceName  string         `gorm:"type:varchar(255)" json:"device_name"`
	DeviceType  string         `gorm:"type:varchar(50);default:'unknown'" json:"device_type"` // web, mobile, desktop, unknown
	DeviceModel string         `gorm:"type:varchar(255)" json:"device_model"`
	OS          string         `gorm:"type:varchar(100)" json:"os"` // iOS, Android, Windows, macOS, Linux, Web
	OSVersion   string         `gorm:"type:varchar(50)" json:"os_version"`
	AppVersion  string         `gorm:"type:varchar(50)" json:"app_version"`
	LastActive  time.Time      `gorm:"autoUpdateTime" json:"last_active"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for GORM
func (DeviceRegistry) TableName() string {
	return "device_registry"
}

// CreateDeviceRequest represents the request body for creating a device
type CreateDeviceRequest struct {
	DeviceName  string `json:"device_name" validate:"omitempty,max=255"`
	DeviceType  string `json:"device_type" validate:"omitempty,oneof=web mobile desktop unknown"`
	DeviceModel string `json:"device_model" validate:"omitempty,max=255"`
	OS          string `json:"os" validate:"omitempty,max=100"`
	OSVersion   string `json:"os_version" validate:"omitempty,max=50"`
	AppVersion  string `json:"app_version" validate:"omitempty,max=50"`
}

// DeviceResponse represents the response for device information
type DeviceResponse struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	DeviceName  string    `json:"device_name"`
	DeviceType  string    `json:"device_type"`
	DeviceModel string    `json:"device_model"`
	OS          string    `json:"os"`
	OSVersion   string    `json:"os_version"`
	AppVersion  string    `json:"app_version"`
	LastActive  time.Time `json:"last_active"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Create creates a new device registry record
func (d *DeviceRegistry) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, d)
	if err != nil {
		return err
	}
	return nil
}

// Update updates an existing device registry record
func (d *DeviceRegistry) Update(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, d)
	return err
}

// GetDeviceByID retrieves a device by its ID
func (d *DeviceRegistry) GetDeviceByID(db *gorm.DB, deviceID string) (DeviceRegistry, error) {
	var device DeviceRegistry

	err := db.Where("id = ?", deviceID).First(&device).Error
	if err != nil {
		return device, err
	}

	return device, nil
}

// GetDevicesByUserID retrieves all devices for a user
func (d *DeviceRegistry) GetDevicesByUserID(db *gorm.DB, userID string) ([]DeviceRegistry, error) {
	var devices []DeviceRegistry

	err := db.Where("user_id = ?", userID).Order("last_active DESC").Find(&devices).Error
	if err != nil {
		return devices, err
	}

	return devices, nil
}

// GetActiveDevicesByUserID retrieves all active devices for a user
func (d *DeviceRegistry) GetActiveDevicesByUserID(db *gorm.DB, userID string) ([]DeviceRegistry, error) {
	var devices []DeviceRegistry

	err := db.Where("user_id = ? AND is_active = ?", userID, true).Order("last_active DESC").Find(&devices).Error
	if err != nil {
		return devices, err
	}

	return devices, nil
}

// DeviceExists checks if a device exists
func (d *DeviceRegistry) DeviceExists(db *gorm.DB, deviceID string) bool {
	return postgresql.CheckExists(db, d, "id = ?", deviceID)
}

// ToResponse converts DeviceRegistry to DeviceResponse
func (d *DeviceRegistry) ToResponse() DeviceResponse {
	return DeviceResponse{
		ID:          d.ID,
		UserID:      d.UserID.String(),
		DeviceName:  d.DeviceName,
		DeviceType:  d.DeviceType,
		DeviceModel: d.DeviceModel,
		OS:          d.OS,
		OSVersion:   d.OSVersion,
		AppVersion:  d.AppVersion,
		LastActive:  d.LastActive,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
