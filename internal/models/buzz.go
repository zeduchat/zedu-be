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
	BuzzParticipantStatusActive   = "active"
	BuzzParticipantStatusLeft     = "left"
	BuzzParticipantStatusInactive = "inactive"
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
	ChannelID string `json:"channel_id" validate:"required,uuid"`
}

// ParticipantMetadata contains detailed information about a buzz participant
type ParticipantMetadata struct {
	UserID    string 	 `json:"user_id"`
	UserName  string 	 `json:"username"`
	FullName  string 	 `json:"full_name"`
	AvatarURL *string 	 `json:"avatar_url,omitempty"`
	JoinedAt  time.Time  `json:"joined_at"`
	Status    string 	 `json:"status"`
}

type BuzzCreateResponse struct {
	BuzzID       string                  `json:"buzz_id"`
	HostID       string                  `json:"host_id"`
	ChannelID    string                  `json:"channel_id"`
	Status       string                  `json:"status"`
	CreatedAt    time.Time               `json:"created_at"`
	StartedAt    time.Time               `json:"started_at"`
	Participants []ParticipantMetadata   `json:"participants"`
	AgoraToken   *BuzzAgoraTokenResponse `json:"agora_token"`
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
	BuzzID       string                  `json:"Buzz_id"`
	ChannelID    string                  `json:"channel_id"`
	UserID       string                  `json:"user_id"`
	Status       string                  `json:"status"`
	JoinedAt     time.Time               `json:"joined_at"`
	Participants []ParticipantMetadata   `json:"participants"`
	AgoraToken   *BuzzAgoraTokenResponse `json:"agora_token"`
}

// Buzz Invitation Models and helper functions

type BuzzInvitationStatus string

const (
	BuzzInvitationPending  BuzzInvitationStatus = "pending"
	BuzzInvitationAccepted BuzzInvitationStatus = "accepted"
	BuzzInvitationDeclined BuzzInvitationStatus = "declined"
	BuzzInvitationExpired  BuzzInvitationStatus = "expired"
)

type BuzzInvitation struct {
	ID          string               `gorm:"type:uuid;primaryKey" json:"id"`
	BuzzID      string               `gorm:"type:uuid;not null;index:idx_buzz_invitations_buzz" json:"buzz_id"`
	ChannelID   string               `gorm:"type:uuid;not null;index" json:"channel_id"`
	InviterID   string               `gorm:"type:uuid;not null" json:"inviter_id"`
	InviteeID   string               `gorm:"type:uuid;not null;index:idx_buzz_invitations_invitee" json:"invitee_id"`
	Status      BuzzInvitationStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	InvitedAt   time.Time            `gorm:"not null" json:"invited_at"`
	RespondedAt *time.Time           `json:"responded_at,omitempty"`
	CreatedAt   time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time            `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Buzz    Buzz    `gorm:"foreignKey:BuzzID;references:ID" json:"-"`
	Inviter Profile `gorm:"foreignKey:InviterID;references:UserID" json:"-"`
	Invitee Profile `gorm:"foreignKey:InviteeID;references:UserID" json:"-"`
}

func (BuzzInvitation) TableName() string {
	return "buzz_invitations"
}

type InviteUsersToBuzzRequest struct {
	BuzzID     string   `json:"buzz_id" validate:"required,uuid"`
	InviteeIDs []string `json:"invitee_ids" validate:"required,min=1,dive,uuid"`
}

type SearchChannelMembersRequest struct {
	ChannelID string `json:"channel_id" validate:"required,uuid"`
	Query     string `json:"query"`
	BuzzID    string `json:"buzz_id" validate:"required,uuid"`
	Limit     int    `json:"limit" validate:"omitempty,min=1,max=100"`
}

type ChannelMemberInfo struct {
	UserID    string  `json:"user_id"`
	UserName  string  `json:"username"`
	FullName  string  `json:"full_name"`
	Email     string  `json:"email"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	IsInBuzz  bool    `json:"is_in_buzz"`
	IsInvited bool    `json:"is_invited"`
}

type SearchChannelMembersResponse struct {
	Members []ChannelMemberInfo `json:"members"`
	Total   int                 `json:"total"`
}

type InviteUsersToBuzzResponse struct {
	BuzzID          string   `json:"buzz_id"`
	InvitedUserIDs  []string `json:"invited_user_ids"`
	FailedUserIDs   []string `json:"failed_user_ids,omitempty"`
	InvitationsSent int      `json:"invitations_sent"`
	Message         string   `json:"message"`
}

type BuzzInvitationResponse struct {
	InvitationID string               `json:"invitation_id"`
	BuzzID       string               `json:"buzz_id"`
	ChannelID    string               `json:"channel_id"`
	InviterID    string               `json:"inviter_id"`
	InviterName  string               `json:"inviter_name"`
	Status       BuzzInvitationStatus `json:"status"`
	InvitedAt    time.Time            `json:"invited_at"`
}

type RespondToInvitationRequest struct {
	InvitationID string `json:"invitation_id" validate:"required,uuid"`
	Accept       bool   `json:"accept"`
}

type RespondToInvitationResponse struct {
	InvitationID string                  `json:"invitation_id"`
	BuzzID       string                  `json:"buzz_id"`
	Status       BuzzInvitationStatus    `json:"status"`
	Message      string                  `json:"message"`
	AgoraToken   *BuzzAgoraTokenResponse `json:"agora_token,omitempty"`
}

type BuzzInvitationEventPayload struct {
	Event        string    `json:"event"`
	InvitationID string    `json:"invitation_id"`
	BuzzID       string    `json:"buzz_id"`
	ChannelID    string    `json:"channel_id"`
	InviterID    string    `json:"inviter_id"`
	InviterName  string    `json:"inviter_name"`
	InviteeID    string    `json:"invitee_id"`
	InvitedAt    time.Time `json:"invited_at"`
}

type BuzzInvitationResponseEventPayload struct {
	Event        string               `json:"event"`
	InvitationID string               `json:"invitation_id"`
	BuzzID       string               `json:"buzz_id"`
	ChannelID    string               `json:"channel_id"`
	InviteeID    string               `json:"invitee_id"`
	InviteeName  string               `json:"invitee_name"`
	Status       BuzzInvitationStatus `json:"status"`
	RespondedAt  time.Time            `json:"responded_at"`
}

// ExpireInvitationsForBuzz marks all pending invitations as expired when a buzz ends
func ExpireInvitationsForBuzz(db *gorm.DB, buzzID string) error {
	return db.Model(&BuzzInvitation{}).
		Where("buzz_id = ? AND status = ?", buzzID, BuzzInvitationPending).
		Updates(map[string]interface{}{
			"status":     BuzzInvitationExpired,
			"updated_at": time.Now().UTC(),
		}).Error
}

// GetPendingInvitationsForUser retrieves all pending invitations for a user
func GetPendingInvitationsForUser(db *gorm.DB, userID string) ([]BuzzInvitation, error) {
	var invitations []BuzzInvitation
	err := db.Where("invitee_id = ? AND status = ?", userID, BuzzInvitationPending).
		Preload("Buzz").
		Preload("Inviter").
		Order("invited_at DESC").
		Find(&invitations).Error
	return invitations, err
}

// CheckInvitationExists checks if a pending invitation already exists
func CheckInvitationExists(db *gorm.DB, buzzID, inviteeID string) (bool, error) {
	var count int64
	err := db.Model(&BuzzInvitation{}).
		Where("buzz_id = ? AND invitee_id = ? AND status = ?", buzzID, inviteeID, BuzzInvitationPending).
		Count(&count).Error
	return count > 0, err
}
