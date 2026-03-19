package models

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

const (
	BuzzStatusActive = "active"
	BuzzStatusEnded  = "ended"
)

const (
	RecordingStatusIdle      = "idle"
	RecordingStatusStarting  = "starting"
	RecordingStatusRecording = "recording"
	RecordingStatusStopping  = "stopping"
	RecordingStatusStopped   = "stopped"
	RecordingStatusFailed    = "failed"
)

const (
	BuzzParticipantStatusActive   = "active"
	BuzzParticipantStatusLeft     = "left"
	BuzzParticipantStatusInactive = "inactive"
)

const (
	ChannelTypeRegular = "channel"
	ChannelTypeDM      = "dm_channel"
	ChannelTypeGroupDM = "group_dm_channel"
)

const (
	BuzzTypeChannel      = "channel"
	BuzzTypeOrganization = "organization"
)

type Buzz struct {
	ID             string         `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID      string         `gorm:"type:uuid;not null;index" json:"channel_id"`
	ChannelType    string         `gorm:"type:varchar(20);not null;default:'channel'" json:"channel_type"`
	HostID         string         `gorm:"type:uuid;not null;index" json:"host_id"`
	OriginalHostID string         `gorm:"type:uuid;index" json:"original_host_id"`
	OrgID          *string        `gorm:"type:uuid;index" json:"org_id,omitempty"`
	BuzzType       string         `gorm:"type:varchar(20);not null;default:'channel'" json:"buzz_type"`
	ParticipantIDs pq.StringArray `gorm:"column:participants;type:text[];not null" json:"participant_ids"`
	BuzzStartTime  time.Time      `gorm:"column:buzz_start_time;autoCreateTime" json:"Buzz_start_time"`
	BuzzEndTime    *time.Time     `gorm:"column:buzz_end_time" json:"Buzz_end_time"`
	IsLiveStatus   bool           `gorm:"column:is_live_status;default:true" json:"is_live_status"`
	Status         string         `gorm:"type:text;default:'active'" json:"status"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type BuzzRecording struct {
	ID            string     `gorm:"type:uuid;primaryKey" json:"id"`
	BuzzID        string     `gorm:"type:uuid;not null;index" json:"buzz_id"`
	OrgID         string     `gorm:"type:uuid;not null;index" json:"org_id"`
	ResourceID    string     `gorm:"type:text;not null" json:"resource_id"`
	Sid           string     `gorm:"type:text;not null" json:"sid"`
	RecorderToken string     `gorm:"type:text" json:"recorder_token"`
	Status        string     `gorm:"type:varchar(20);not null;default:'starting'" json:"status"`
	FileURL       string     `gorm:"type:text" json:"file_url"`
	FileID        *string    `gorm:"type:uuid" json:"file_id,omitempty"`
	DurationSec   int        `gorm:"default:0" json:"duration_sec"`
	StartedAt     time.Time  `gorm:"not null" json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type BuzzParticipant struct {
	ID            string     `gorm:"type:uuid;primaryKey" json:"id"`
	BuzzID        string     `gorm:"type:uuid;index;not null" json:"buzz_id"`
	UserID        string     `gorm:"type:uuid;index;not null" json:"user_id"`
	Status        string     `gorm:"type:text;not null;default:'active'" json:"status"`
	IsMuted       bool       `gorm:"type:boolean;default:false" json:"is_muted"`
	StatusSticker *string    `gorm:"type:varchar(50)" json:"status_sticker,omitempty"`
	StickerSetAt  *time.Time `gorm:"column:sticker_set_at" json:"sticker_set_at,omitempty"`
	MediaState    *string    `gorm:"type:jsonb" json:"media_state,omitempty"`
	JoinedAt      time.Time  `gorm:"column:joined_at;not null;autoCreateTime" json:"joined_at"`
	LeftAt        *time.Time `gorm:"column:left_at" json:"left_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type CreateBuzzRequest struct {
	ChannelID     string  `json:"channel_id" validate:"omitempty,uuid"`
	ParticipantID *string `json:"participant_id" validate:"omitempty,uuid"`
}

// ParticipantMetadata contains detailed information about a buzz participant
type ParticipantMetadata struct {
	UserID        string     `json:"user_id"`
	UserName      string     `json:"username"`
	FullName      string     `json:"full_name"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	JoinedAt      time.Time  `json:"joined_at"`
	Status        string     `json:"status"`
	JoinStatus    string     `json:"join_status"`
	StatusSticker *string    `json:"status_sticker,omitempty"`
	StickerSetAt  *time.Time `json:"sticker_set_at,omitempty"`
	MediaState    *string    `json:"media_state,omitempty"`
}

type BuzzCreateResponse struct {
	BuzzID          string                  `json:"buzz_id"`
	HostID          string                  `json:"host_id"`
	ChannelID       string                  `json:"channel_id"`
	Status          string                  `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	StartedAt       time.Time               `json:"started_at"`
	EndedAt         *time.Time              `json:"ended_at,omitempty"`
	ParticipantIDs  []string                `json:"participants_id"`
	Participants    []ParticipantMetadata   `json:"participants"`
	AgoraToken      *BuzzAgoraTokenResponse `json:"agora_token"`
	BuzzCode        string                  `json:"buzz_code"`
	IsRecording     bool                    `json:"is_recording"`
	RecordingStatus string                  `json:"recording_status"`
}

type BuzzLeaveResponse struct {
	BuzzID        string    `json:"buzz_id"`
	BuzzCode      string    `json:"buzz_code"`
	ParticipantID string    `json:"participant_id"`
	NewHostID     string    `json:"new_host_id,omitempty"`
	LeftAt        time.Time `json:"left_at"`
	BuzzEnded     bool      `json:"buzz_ended"`
}

type BuzzEndResponse struct {
	BuzzID    string    `json:"buzz_id"`
	BuzzCode  string    `json:"buzz_code"`
	ChannelID string    `json:"channel_id"`
	HostID    string    `json:"host_id"`
	EndedAt   time.Time `json:"ended_at"`
	Status    string    `json:"status"`
}

type BuzzMetadataResponse struct {
	BuzzID          string                  `json:"buzz_id"`
	BuzzCode        string                  `json:"buzz_code"`
	HostID          string                  `json:"host_id"`
	ChannelID       string                  `json:"channel_id"`
	Status          string                  `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	StartedAt       time.Time               `json:"started_at"`
	EndedAt         *time.Time              `json:"ended_at,omitempty"`
	Participants    []ParticipantMetadata   `json:"participants"`
	AgoraToken      *BuzzAgoraTokenResponse `json:"agora_token"`
	HostName        string                  `json:"host_name"`
	RecordingStatus string                  `json:"recording_status"`
	IsRecording     bool                    `json:"is_recording"`
	LastJoinedUser  *ParticipantMetadata    `json:"last_joined_user,omitempty"`
}

type ActiveBuzzIndicator struct {
	IsActive              bool       `json:"is_active"`
	BuzzID                string     `json:"buzz_id,omitempty"`
	BuzzCode              string     `json:"buzz_code,omitempty"`
	HostID                string     `json:"host_id,omitempty"`
	Status                string     `json:"status,omitempty"`
	EndedAt               *time.Time `json:"ended_at,omitempty"`
	ParticipantCount      int        `json:"participant_count"`
	ParticipantPreview    []string   `json:"participant_preview"`
	RemainingParticipants int        `json:"remaining_participants"`
	IsUserInBuzz          bool       `json:"is_user_in_buzz,omitempty"`
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

func (h *Buzz) GetRemainingTime(maxDurationInSeconds int) uint32 {
	elapsed := time.Since(h.CreatedAt).Seconds()
	remaining := float64(maxDurationInSeconds) - elapsed
	if remaining < 0 {
		return 0
	}
	return uint32(remaining)
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
	// Check regular channels with explicit error handling
	var userChannel UserChannels
	err := db.Where("channels_id = ? AND user_id = ?", channelID, userID).First(&userChannel).Error
	if err == nil {
		return true
	}

	// Check DM channels - user can be either the creator (user_id) OR the participant (participant_id)
	var dmChannel DmChannels
	err = db.Where("channel_id = ? AND (user_id = ? OR participant_id = ?)", channelID, userID, userID).First(&dmChannel).Error
	if err == nil {
		return true
	}

	// Check group DM channels
	var participant ChannelParticipant
	err = db.Where("channel_id = ? AND user_id = ?", channelID, userID).First(&participant).Error
	if err == nil {
		return true
	}

	return false
}

type BuzzEventPayload struct {
	Event              string               `json:"event"`
	BuzzID             string               `json:"buzz_id"`
	ChannelID          string               `json:"channel_id"`
	HostID             string               `json:"host_id"`
	ParticipantIDs     []string             `json:"participant_ids",omitempty`
	ParticipantDetails []ParticipantDetails `json:"participant_details,omitempty"`
	CreatedAt          time.Time            `json:"created_at",omitempty`
	Status             string               `json:"status"`
	UserJoined         ParticipantDetails   `json:"user_joined,omitempty"`
	UserLeft           ParticipantDetails   `json:"user_left,omitempty"`
	IsRecording        bool                 `json:"is_recording"`
	RecordingStatus    string               `json:"recording_status"`
}

type ParticipantDetails struct {
	UserID     string  `json:"user_id"`
	Username   string  `json:"username"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
	Email      string  `json:"email,omitempty"`
	JoinStatus string  `json:"join_status,omitempty"`
	MediaState *string `json:"media_state,omitempty"`
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
	BuzzID    string    `gorm:"type:uuid;not null;index" json:"buzz_id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Note      string    `gorm:"type:text;not null" json:"note"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type CreateBuzzNoteRequest struct {
	Note string `json:"note" validate:"required"`
}

type SendBuzzMessageRequest struct {
	Content string `json:"content" validate:"required"`
	Media   []File `json:"media"`
}

type UpdateBuzzNoteRequest struct {
	Note string `json:"note" validate:"required"`
}

type BuzzNoteResponse struct {
	ID        string    `json:"id"`
	BuzzID    string    `json:"buzz_id"`
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
	BuzzID     string `json:"buzz_id"`
	UserID     string `json:"user_id"`
	IsCameraOn bool   `json:"is_camera_on"`
	UpdatedAt  string `json:"updated_at"`
}

type CameraStatusEventPayload struct {
	Event      string `json:"event"`
	BuzzID     string `json:"buzz_id"`
	ChannelID  string `json:"channel_id"`
	UserID     string `json:"user_id"`
	IsCameraOn bool   `json:"is_camera_on"`
	Timestamp  string `json:"timestamp"`
}

type UpdateMediaStateRequest struct {
	MediaState interface{} `json:"media_state" validate:"required"`
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

	if h.BuzzType != BuzzTypeOrganization {
		if !IsUserInChannel(db, h.ChannelID, userID) {
			return errors.New("user is not a member of the channel")
		}
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
			return nil
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
	BuzzID string `json:"buzz_id" validate:"required,uuid"`
}

// JoinBuzzResponse represents the response after joining a Buzz
type JoinBuzzResponse struct {
	BuzzID          string                  `json:"buzz_id"`
	BuzzCode        string                  `json:"buzz_code"`
	HostID          string                  `json:"host_id"`
	ChannelID       string                  `json:"channel_id"`
	UserID          string                  `json:"user_id"`
	Status          string                  `json:"status"`
	CreatedAt       time.Time               `json:"created_at"`
	StartedAt       time.Time               `json:"started_at"`
	EndedAt         *time.Time              `json:"ended_at,omitempty"`
	JoinedAt        time.Time               `json:"joined_at"`
	Participants    []ParticipantMetadata   `json:"participants"`
	AgoraToken      *BuzzAgoraTokenResponse `json:"agora_token"`
	HostRestored    bool                    `json:"host_restored,omitempty"`
	IsRecording     bool                    `json:"is_recording"`
	RecordingStatus string                  `json:"recording_status"`
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
	Inviter Profile `gorm:"foreignKey:InviterID;references:Userid" json:"-"`
	Invitee Profile `gorm:"foreignKey:InviteeID;references:Userid" json:"-"`
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

// BuzzReactionPayload represents an ephemeral reaction event (not persisted, real-time only)
type BuzzReactionPayload struct {
	Event        string    `json:"event"` // Always "buzz_reaction"
	BuzzID       string    `json:"buzz_id"`
	ChannelID    string    `json:"channel_id"`
	UserID       string    `json:"user_id"`
	UserName     string    `json:"username"`
	ReactionType string    `json:"reaction_type"` // "emoji", "effect", "gif"
	Content      string    `json:"content"`       // emoji code, effect name, or gif URL
	Timestamp    time.Time `json:"timestamp"`
}

// SendBuzzReactionRequest represents the request to send an ephemeral reaction
type SendBuzzReactionRequest struct {
	BuzzID       string `json:"buzz_id" validate:"required,uuid"`
	ReactionType string `json:"reaction_type" validate:"required,oneof=emoji effect gif"`
	Content      string `json:"content" validate:"required"`
}

// BuzzStickerUpdateRequest represents the request to update a status sticker
type BuzzStickerUpdateRequest struct {
	BuzzID  string  `json:"buzz_id" validate:"required,uuid"`
	Sticker *string `json:"sticker" validate:"omitempty,oneof=raise_hand brb away"` // null to clear
}

// BuzzStickerUpdateResponse represents the response after updating a sticker
type BuzzStickerUpdateResponse struct {
	BuzzID       string     `json:"buzz_id"`
	UserID       string     `json:"user_id"`
	Sticker      *string    `json:"sticker"`
	StickerSetAt *time.Time `json:"sticker_set_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// BuzzStickerPayload represents a status sticker change event (broadcast via Centrifuge)
type BuzzStickerPayload struct {
	Event        string     `json:"event"` // "buzz_sticker_update"
	BuzzID       string     `json:"buzz_id"`
	ChannelID    string     `json:"channel_id"`
	UserID       string     `json:"user_id"`
	UserName     string     `json:"username"`
	Sticker      *string    `json:"sticker"`
	StickerSetAt *time.Time `json:"sticker_set_at,omitempty"`
	Timestamp    time.Time  `json:"timestamp"`
}

type OrgBuzzListResponse struct {
	Buzzes []OrgBuzzItem `json:"buzzes"`
	Total  int           `json:"total"`
}

type OrgBuzzItem struct {
	BuzzID           string     `json:"buzz_id"`
	BuzzCode         string     `json:"buzz_code"`
	ChannelID        string     `json:"channel_id"`
	HostID           string     `json:"host_id"`
	OrgID            string     `json:"org_id"`
	Status           string     `json:"status"`
	ParticipantCount int        `json:"participant_count"`
	CreatedAt        time.Time  `json:"created_at"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
}

type BuzzTimeWarningPayload struct {
	Event            string    `json:"event"`
	BuzzID           string    `json:"buzz_id"`
	ChannelID        string    `json:"channel_id"`
	HostID           string    `json:"host_id"`
	RemainingMinutes int       `json:"remaining_minutes"`
	EstimatedEndTime time.Time `json:"estimated_end_time"`
	Timestamp        time.Time `json:"timestamp"`
}

const (
	DirectCallRingingTimeoutMinutes = 5

	CallStatusPending  = "pending"
	CallStatusAccepted = "accepted"
	CallStatusDeclined = "declined"
	CallStatusTimeout  = "timeout"
)

type DirectCallParticipant struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	AvatarURL        string `json:"avatar_url"`
	DefaultAvatarURL string `json:"default_avatar_url"`
	JoinStatus       string `json:"join_status"`
}

type InitiateDirectCallRequest struct {
	ChannelID string `json:"channel_id" validate:"required,uuid"`
}

type RespondToCallRequest struct {
	Action string `json:"action" validate:"required,oneof=accept decline timeout cancel"`
}

type DirectCallResponse struct {
	BuzzID       string                  `json:"buzz_id"`
	BuzzCode     string                  `json:"buzz_code"`
	ChannelID    string                  `json:"channel_id"`
	CallerID     string                  `json:"caller_id"`
	CallerName   string                  `json:"caller_name"`
	Status       string                  `json:"status"`
	JoinStatus   string                  `json:"join_status"`
	Participants []DirectCallParticipant `json:"participants"`
	UserJoined   *DirectCallParticipant  `json:"user_joined,omitempty"`
	UserRejected *DirectCallParticipant  `json:"user_rejected,omitempty"`
	UserTimeout  *DirectCallParticipant  `json:"user_timeout,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	AgoraToken   *BuzzAgoraTokenResponse `json:"agora_token,omitempty"`
}

type CallPushPayload struct {
	BuzzID           string `json:"buzz_id"`
	ChannelID        string `json:"channel_id"`
	CallerName       string `json:"caller_name"`
	CallerID         string `json:"caller_id"`
	AvatarURL        string `json:"avatar_url"`
	DefaultAvatarURL string `json:"default_avatar_url"`
	Event            string `json:"event"`
}

type DirectCallCentrifugoPayload struct {
	Event        string                  `json:"event"`
	BuzzID       string                  `json:"buzz_id"`
	ChannelID    string                  `json:"channel_id"`
	CallerID     string                  `json:"caller_id"`
	CallerName   string                  `json:"caller_name"`
	JoinStatus   string                  `json:"join_status"`
	Participants []DirectCallParticipant `json:"participants"`
	UserJoined   *DirectCallParticipant  `json:"user_joined,omitempty"`
	UserRejected *DirectCallParticipant  `json:"user_rejected,omitempty"`
	UserTimeout  *DirectCallParticipant  `json:"user_timeout,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
}

func GetDMParticipants(db *gorm.DB, channelID string) ([]string, error) {
	var dmChannel DmChannels
	if err := db.Where("channel_id = ?", channelID).First(&dmChannel).Error; err != nil {
		return nil, errors.New("channel not found")
	}

	if dmChannel.ChannelType == "group_dm" {
		var participants []ChannelParticipant
		if err := db.Where("channel_id = ? AND deleted_at IS NULL", channelID).Find(&participants).Error; err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(participants))
		for _, p := range participants {
			ids = append(ids, p.UserId)
		}
		return ids, nil
	}

	var allDMs []DmChannels
	if err := db.Where("channel_id = ?", channelID).Find(&allDMs).Error; err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	ids := []string{}
	for _, dm := range allDMs {
		if !seen[dm.UserId] {
			seen[dm.UserId] = true
			ids = append(ids, dm.UserId)
		}
		if dm.ParticipantId != nil && !seen[*dm.ParticipantId] {
			seen[*dm.ParticipantId] = true
			ids = append(ids, *dm.ParticipantId)
		}
	}
	return ids, nil
}

func GetDMChannelType(db *gorm.DB, channelID string) (string, error) {
	var dmChannel DmChannels
	if err := db.Where("channel_id = ?", channelID).First(&dmChannel).Error; err != nil {
		return "", errors.New("channel not found")
	}
	return dmChannel.ChannelType, nil
}

func (b *Buzz) AppendParticipant(db *gorm.DB, userID string) error {
	return db.Exec(
		"UPDATE buzzs SET participants = array_append(participants, ?) WHERE id = ? AND NOT (? = ANY(participants))",
		userID, b.ID, userID,
	).Error
}
