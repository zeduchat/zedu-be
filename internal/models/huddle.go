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

type Huddle struct {
	ID              string         `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID       string         `gorm:"type:uuid;not null;index" json:"channel_id"`
	HostID          string         `gorm:"type:uuid;not null;index" json:"host_id"`
	Participants    pq.StringArray `gorm:"type:text[];not null" json:"participants"`
	HuddleStartTime time.Time      `gorm:"column:huddle_start_time;autoCreateTime" json:"huddle_start_time"`
	HuddleEndTime   *time.Time     `gorm:"column:huddle_end_time" json:"huddle_end_time"`
	IsLiveStatus    bool           `gorm:"column:is_live_status;default:true" json:"is_live_status"`
	Status          string         `gorm:"type:text;default:'active'" json:"status"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

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
	HuddleID     string    `json:"huddle_id"`
	HostID       string    `json:"host_id"`
	ChannelID    string    `json:"channel_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	StartedAt    time.Time `json:"started_at"`
	Participants []string  `json:"participants"`
}

func (h *Huddle) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = utility.GenerateUUID()
	}
	if h.Status == "" {
		h.Status = HuddleStatusActive
	}
	if len(h.Participants) == 0 {
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
	Event        string    `json:"event"`
	HuddleID     string    `json:"huddle_id"`
	ChannelID    string    `json:"channel_id"`
	HostID       string    `json:"host_id"`
	Participants []string  `json:"participants"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}
