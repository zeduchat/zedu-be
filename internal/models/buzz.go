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
	BuzzStatusActive = "active"
	BuzzStatusEnded  = "ended"
)

const (
	BuzzParticipantStatusActive = "active"
	BuzzParticipantStatusLeft   = "left"
)

type Buzz struct {
	ID             string         `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID      string         `gorm:"type:uuid;not null;index" json:"channel_id"`
	HostID         string         `gorm:"type:uuid;not null;index" json:"host_id"`
	ParticipantIDs pq.StringArray `gorm:"column:participants;type:text[];not null" json:"participant_ids"`
	BuzzStartTime  time.Time      `gorm:"column:Buzz_start_time;autoCreateTime" json:"Buzz_start_time"`
	BuzzEndTime    *time.Time     `gorm:"column:Buzz_end_time" json:"Buzz_end_time"`
	IsLiveStatus   bool           `gorm:"column:is_live_status;default:true" json:"is_live_status"`
	Status         string         `gorm:"type:text;default:'active'" json:"status"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type BuzzParticipant struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	BuzzID    string     `gorm:"type:uuid;index;not null" json:"Buzz_id"`
	UserID    string     `gorm:"type:uuid;index;not null" json:"user_id"`
	Status    string     `gorm:"type:text;not null;default:'active'" json:"status"`
	IsMuted   bool       `gorm:"type:boolean;default:false" json:"is_muted"`
	JoinedAt  time.Time  `gorm:"column:joined_at;not null;autoCreateTime" json:"joined_at"`
	LeftAt    *time.Time `gorm:"column:left_at" json:"left_at"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type CreateBuzzRequest struct {
	ChannelID      string   `json:"channel_id" validate:"required,uuid"`
	ParticipantIDs []string `json:"participant_ids"`
	OrganisationID string   `json:"organisation_id,omitempty"`
}

type BuzzCreateResponse struct {
	BuzzID         string    `json:"buzz_id"`
	HostID         string    `json:"host_id"`
	ChannelID      string    `json:"channel_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      time.Time `json:"started_at"`
	ParticipantIDs []string  `json:"participant_ids"`
}

type BuzzLeaveResponse struct {
	BuzzID        string    `json:"buzz_id"`
	ParticipantID string    `json:"participant_id"`
	NewHostID     string    `json:"new_host_id,omitempty"`
	LeftAt        time.Time `json:"left_at"`
	BuzzEnded     bool      `json:"buzz_ended"`
}

func (h *Buzz) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = utility.GenerateUUID()
	}
	if h.Status == "" {
		h.Status = BuzzStatusActive
	}
	if len(h.ParticipantIDs) == 0 {
		return errors.New("participants cannot be empty")
	}
	return nil
}

func (hp *BuzzParticipant) BeforeCreate(tx *gorm.DB) error {
	if hp.ID == "" {
		hp.ID = utility.GenerateUUID()
	}
	if hp.Status == "" {
		hp.Status = BuzzParticipantStatusActive
	}
	return nil
}

func IsUserInChannel(db *gorm.DB, channelID, userID string) bool {
	return postgresql.CheckExists(db, &UserChannels{}, "channels_id = ? AND user_id = ?", channelID, userID)
}

type BuzzEventPayload struct {
	Event          string    `json:"event"`
	BuzzID         string    `json:"Buzz_id"`
	ChannelID      string    `json:"channel_id"`
	HostID         string    `json:"host_id"`
	ParticipantIDs []string  `json:"participant_ids"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
}

type BuzzLeaveEventPayload struct {
	UserID       string `json:"user_id"`
	UserName     string `json:"user_name"`
	HuddleStatus string `json:"huddle_status"`
	HostChanged  bool   `json:"host_changed"`
	BuzzEventPayload
}

type BuzzNote struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	BuzzID    string    `gorm:"type:uuid;not null;index" json:"Buzz_id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Note      string    `gorm:"type:text;not null" json:"note"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type CreateBuzzNoteRequest struct {
	Note string `json:"note" validate:"required"`
}

type UpdateBuzzNoteRequest struct {
	Note string `json:"note" validate:"required"`
}

type BuzzNoteResponse struct {
	ID        string    `json:"id"`
	BuzzID    string    `json:"Buzz_id"`
	UserID    string    `json:"user_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BuzzNotesListResponse struct {
	Notes []BuzzNoteResponse `json:"notes"`
}

type UpdateCameraRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
	Status bool   `json:"status" validate:"required"`
}

type UpdateCameraResponse struct {
	BuzzID     string `json:"Buzz_id"`
	UserID     string `json:"user_id"`
	IsCameraOn bool   `json:"is_camera_on"`
	UpdatedAt  string `json:"updated_at"`
}

type CameraStatusEventPayload struct {
	Event      string `json:"event"`
	BuzzID     string `json:"Buzz_id"`
	ChannelID  string `json:"channel_id"`
	UserID     string `json:"user_id"`
	IsCameraOn bool   `json:"is_camera_on"`
	Timestamp  string `json:"timestamp"`
}

// AddUserToBuzz adds a user to a Buzz as a participant
func (h *Buzz) AddUserToBuzz(db *gorm.DB, userID string) error {
	// Validate inputs
	if h.ID == "" {
		return errors.New("Buzz does not exist")
	}
	if userID == "" {
		return errors.New("invalid user ID")
	}

	// Check if user is in the channel
	if !IsUserInChannel(db, h.ChannelID, userID) {
		return errors.New("user is not a member of the channel")
	}

	// Check if Buzz exists
	var Buzz Buzz
	if err := db.Where("id = ?", h.ID).First(&Buzz).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("Buzz does not exist")
		}
		return err
	}

	// Check if Buzz is active
	if Buzz.Status != BuzzStatusActive {
		return errors.New("Buzz is not active")
	}

	// Check if user is already in the Buzz
	for _, participantID := range Buzz.ParticipantIDs {
		if participantID == userID {
			return errors.New("user is already in the Buzz")
		}
	}

	// Add user to participants array
	return updateBuzzParticipants(db, h.ID, append(Buzz.ParticipantIDs, userID))
}

// updateBuzzParticipants updates the participants array of a Buzz
func updateBuzzParticipants(db *gorm.DB, BuzzID string, participants pq.StringArray) error {
	return db.Model(&Buzz{}).
		Where("id = ?", BuzzID).
		Update("participants", participants).Error
}

// JoinBuzzRequest represents the request to join a Buzz
type JoinBuzzRequest struct {
	BuzzID string `json:"Buzz_id" validate:"required,uuid"`
}

// JoinBuzzResponse represents the response after joining a Buzz
type JoinBuzzResponse struct {
	BuzzID         string    `json:"Buzz_id"`
	ChannelID      string    `json:"channel_id"`
	UserID         string    `json:"user_id"`
	Status         string    `json:"status"`
	JoinedAt       time.Time `json:"joined_at"`
	ParticipantIDs []string  `json:"participant_ids"`
}
