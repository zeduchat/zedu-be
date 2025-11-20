package models

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

const (
	HuddleStatusActive = "active"
	HuddleStatusEnded  = "ended"
)

const (
	HuddleParticipantStatusActive = "active"
	HuddleParticipantStatusLeft   = "left"
)

// Huddle represents a huddle in the system
type Huddle struct {
	ID              string         `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID       string         `gorm:"type:uuid;not null;index" json:"channel_id"`
	HostID          string         `gorm:"type:uuid;not null;index" json:"host_id"`
	ParticipantIDs  pq.StringArray `gorm:"column:participants;type:text[];not null" json:"participant_ids"`
	HuddleStartTime time.Time      `gorm:"column:huddle_start_time;autoCreateTime" json:"huddle_start_time"`
	HuddleEndTime   *time.Time     `gorm:"column:huddle_end_time" json:"huddle_end_time"`
	IsLiveStatus    bool           `gorm:"column:is_live_status;default:true" json:"is_live_status"`
	Status          string         `gorm:"type:text;default:'active'" json:"status"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// HuddleParticipant represents the participants in the huddle
type HuddleParticipant struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	HuddleID  string     `gorm:"type:uuid;index;not null" json:"huddle_id"`
	UserID    string     `gorm:"type:uuid;index;not null" json:"user_id"`
	Status    string     `gorm:"type:text;not null;default:'active'" json:"status"`
	IsMuted   bool       `gorm:"type:boolean;default:false" json:"is_muted"`
	JoinedAt  time.Time  `gorm:"column:joined_at;not null;autoCreateTime" json:"joined_at"`
	LeftAt    *time.Time `gorm:"column:left_at" json:"left_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type CreateHuddleRequest struct {
	ChannelID      string   `json:"channel_id" validate:"required,uuid"`
	ParticipantIDs []string `json:"participant_ids"`
	OrganisationID string   `json:"organisation_id,omitempty"`
}

type HuddleCreateResponse struct {
	HuddleID       string    `json:"huddle_id"`
	HostID         string    `json:"host_id"`
	ChannelID      string    `json:"channel_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      time.Time `json:"started_at"`
	ParticipantIDs []string  `json:"participant_ids"`
}

func (h *Huddle) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = utility.GenerateUUID()
	}
	if h.Status == "" {
		h.Status = HuddleStatusActive
	}
	if len(h.ParticipantIDs) == 0 {
		return errors.New("participants cannot be empty")
	}
	return nil
}

func (hp *HuddleParticipant) BeforeCreate(tx *gorm.DB) error {
	if hp.ID == "" {
		hp.ID = utility.GenerateUUID()
	}
	if hp.Status == "" {
		hp.Status = HuddleParticipantStatusActive
	}
	return nil
}

func IsUserInChannel(db *gorm.DB, channelID, userID string) bool {
	return postgresql.CheckExists(db, &UserChannels{}, "channels_id = ? AND user_id = ?", channelID, userID)
}

type HuddleEventPayload struct {
	Event          string    `json:"event"`
	HuddleID       string    `json:"huddle_id"`
	ChannelID      string    `json:"channel_id"`
	HostID         string    `json:"host_id"`
	ParticipantIDs []string  `json:"participant_ids"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
}

type HuddleNote struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	HuddleID  string    `gorm:"type:uuid;not null;index" json:"huddle_id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Note      string    `gorm:"type:text;not null" json:"note"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type CreateHuddleNoteRequest struct {
	Note string `json:"note" validate:"required"`
}

type UpdateHuddleNoteRequest struct {
	Note string `json:"note" validate:"required"`
}

type HuddleNoteResponse struct {
	ID        string    `json:"id"`
	HuddleID  string    `json:"huddle_id"`
	UserID    string    `json:"user_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HuddleNotesListResponse struct {
	Notes []HuddleNoteResponse `json:"notes"`
}

type UpdateCameraRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Status bool   `json:"status" validate:"required"`
}

type UpdateCameraResponse struct {
	HuddleID   string `json:"huddle_id"`
	UserID     string `json:"user_id"`
	IsCameraOn bool   `json:"is_camera_on"`
	UpdatedAt  string `json:"updated_at"`
}

type CameraStatusEventPayload struct {
	Event      string `json:"event"`
	HuddleID   string `json:"huddle_id"`
	ChannelID  string `json:"channel_id"`
	UserID     string `json:"user_id"`
	IsCameraOn bool   `json:"is_camera_on"`
	Timestamp  string `json:"timestamp"`
}

// AddUserToHuddle adds a user to a huddle as a participant
func (h *Huddle) AddUserToHuddle(db *gorm.DB, userID string) error {
	// Validate inputs
	if h.ID == "" {
		return errors.New("huddle does not exist")
	}
	if userID == "" {
		return errors.New("invalid user ID")
	}

	// Check if user is in the channel
	if !IsUserInChannel(db, h.ChannelID, userID) {
		return errors.New("user is not a member of the channel")
	}

	// Check if huddle exists
	var huddle Huddle
	if err := db.Where("id = ?", h.ID).First(&huddle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("huddle does not exist")
		}
		return err
	}

	// Check if huddle is active
	if huddle.Status != HuddleStatusActive {
		return errors.New("huddle is not active")
	}

	// Check if user is already in the huddle
	for _, participantID := range huddle.ParticipantIDs {
		if participantID == userID {
			return errors.New("user is already in the huddle")
		}
	}

	// Add user to participants array
	return updateHuddleParticipants(db, h.ID, append(huddle.ParticipantIDs, userID))
}

// updateHuddleParticipants updates the participants array of a huddle
func updateHuddleParticipants(db *gorm.DB, huddleID string, participants pq.StringArray) error {
	return db.Model(&Huddle{}).
		Where("id = ?", huddleID).
		Update("participants", participants).Error
}

// JoinHuddleRequest represents the request to join a huddle
type JoinHuddleRequest struct {
	HuddleID string `json:"huddle_id" validate:"required,uuid"`
}

// JoinHuddleResponse represents the response after joining a huddle
type JoinHuddleResponse struct {
	HuddleID       string    `json:"huddle_id"`
	ChannelID      string    `json:"channel_id"`
	UserID         string    `json:"user_id"`
	Status         string    `json:"status"`
	JoinedAt       time.Time `json:"joined_at"`
	ParticipantIDs []string  `json:"participant_ids"`
}
