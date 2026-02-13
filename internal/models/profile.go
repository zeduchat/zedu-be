package models

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type Profile struct {
	ID                string         `gorm:"type:uuid;primary_key" json:"profile_id"`
	FirstName         string         `gorm:"column:first_name; type:text; not null" json:"first_name"`
	LastName          string         `gorm:"column:last_name; type:text;not null" json:"last_name"`
	FullName          string         `gorm:"column:full_name; type:text;" json:"full_name"`
	UserName          string         `gorm:"column:user_name; type:text;" json:"username"`
	Phone             string         `gorm:"type:varchar(255)" json:"phone"`
	AvatarURL         string         `gorm:"type:varchar(255)" json:"avatar_url"`
	Userid            string         `gorm:"type:uuid;unique" json:"user_id"`
	CreatedAt         time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	DisplayName       string         `gorm:"type:varchar(255)" json:"display_name"`
	Title             string         `gorm:"type:varchar(255)" json:"title"`
	NamePronunciation string         `gorm:"type:varchar(255)" json:"name_pronunciation"`
	Timezone          string         `gorm:"type:varchar(255)" json:"timezone"`
	Icon              string         `gorm:"type:varchar(255)" json:"icon"`
	Text              string         `gorm:"type:varchar(255)" json:"text"`
	PauseNotification bool           `gorm:"type:boolean;default:false" json:"pause_notification"`
	StatusTimeout     string         `gorm:"type:varchar(255)" json:"status_timeout"`
	StatusVisibility  string         `gorm:"type:varchar(255);default:'public'" json:"status_visibility"`
	WorkspaceID       string         `gorm:"type:varchar(255)" json:"workspace_id"`
	Track             string         `gorm:"type:varchar(255)" json:"track"`
	Links             pq.StringArray `gorm:"type:text[]" json:"links"`
}

type ProfileSummary struct {
	ID                string   `json:"id"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone"`
	FirstName         string   `json:"first_name"`
	LastName          string   `json:"last_name"`
	FullName          string   `json:"full_name"`
	UserName          string   `json:"username"`
	AvatarURL         string   `json:"avatar_url"`
	UserId            string   `json:"user_id"`
	Deactivated       bool     `json:"deactivated"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	DeletedAt         string   `json:"deleted_at"`
	ProfileUpdated    bool     `json:"profile_updated"`
	IsOnboarded       bool     `json:"is_onboarded"`
	DisplayName       string   `json:"display_name"`
	Title             string   `json:"title"`
	NamePronunciation string   `json:"name_pronounciation"`
	Timezone          string   `json:"timezone"`
	Icon              string   `json:"icon"`
	Text              string   `json:"text"`
	PauseNotification bool     `json:"pause_notification"`
	StatusTimeout     string   `json:"status_timeout"`
	WorkspaceID       string   `json:"workspace_id"`
	Track             string   `json:"track"`
	Links             []string `json:"links"`
}

type UpdateUserProfileRequest struct {
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	FullName          string `json:"full_name"`
	UserName          string `json:"username"`
	AvatarURL         string `json:"avatar_url"`
	AvatarFile        string `json:"avatar_file"`
	DisplayName       string `json:"display_name"`
	Title             string `json:"title"`
	NamePronunciation string `json:"name_pronounciation"`
	Timezone          string `json:"timezone"`
	AvatarUpdate      bool
	WorkspaceID       string   `json:"workspace_id"`
	Track             string   `json:"track"`
	Links             []string `json:"links"`
}

type UpdateProfileStatus struct {
	Icon              string `json:"icon"`
	Text              string `json:"text"`
	PauseNotification bool   `json:"pause_notification"`
	StatusTimeout     string `json:"status_timeout"`
	ClearStatus       bool   `json:"clear_status"`
	UserId            string
	OrgId             string
}

// PartialStatusUpdate holds optional fields for status patch requests.
// Pointer fields let us detect whether a field was supplied so we can
// avoid overwriting existing values.
type PartialStatusUpdate struct {
	Text       *string `json:"text"`
	Emoji      *string `json:"emoji"`
	Expiry     *int64  `json:"expiry"`
	Visibility *string `json:"visibility"`
	UserID     string  `json:"-"`
}

// SetStatusRequest holds fields for setting a new status via POST.
// Text is required; emoji, expiry, and visibility are optional.
type SetStatusRequest struct {
	Text       string  `json:"text" validate:"required,min=1,max=255"`
	Emoji      *string `json:"emoji,omitempty" validate:"omitempty,max=64,no_whitespace,emoji"`
	Expiry     *int64  `json:"expiry,omitempty" validate:"omitempty,min=0"`
	Visibility *string `json:"visibility,omitempty" validate:"omitempty,oneof=public contacts private"`
	UserID     string  `json:"-"`
}

// UserStatus represents the persisted status for responses.
type UserStatus struct {
	Text       string `json:"text"`
	Emoji      string `json:"emoji"`
	Expiry     int64  `json:"expiry"`
	Visibility string `json:"visibility"`
}

func (j *Profile) UpdateProfileFields(db *gorm.DB, req UpdateUserProfileRequest, userId string, logger *utility.Logger) (*Profile, error) {
	var userProfile Profile

	if req.DisplayName != "" && req.UserName == "" {
		req.UserName = req.DisplayName
	}

	profileUpdates := Profile{
		FullName:          req.FullName,
		UserName:          req.UserName,
		Phone:             req.Phone,
		AvatarURL:         req.AvatarURL,
		DisplayName:       req.DisplayName,
		Title:             req.Title,
		NamePronunciation: req.NamePronunciation,
		Timezone:          req.Timezone,
		WorkspaceID:       req.WorkspaceID,
		Track:             req.Track,
		Links:             pq.StringArray(req.Links),
	}

	query := "userid = ?"

	exist := postgresql.CheckExists(db, &userProfile, query, userId)
	if !exist {
		return nil, errors.New("Profile does not exists")
	}

	if req.AvatarUpdate || (req.UserName != "" && req.UserName != userProfile.UserName) {

		updateThreadsIndex := ThreadDocument{
			UserId:    userId,
			Username:  req.UserName,
			AvatarURL: req.AvatarURL,
		}

		logger.Info("Updating username and avatar across threads index")
		go updateThreadsIndex.UpdateThreadUserProfile(logger, &sync.Mutex{})

		updateMessagesIndex := MessageDocument{
			UserID:    userId,
			Username:  req.UserName,
			AvatarURL: req.AvatarURL,
		}

		logger.Info("Updating username and avatar across messages index")
		go updateMessagesIndex.UpdateMessageUserProfile(logger, &sync.Mutex{})

		logger.Info("Successfully updated username and avatar across indexess")
	}

	result, err := postgresql.UpdateFieldsAndReturn(db, &j, profileUpdates, query, userId)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update user profile")
	}

	return j, nil
}

func (j *Profile) UpdateProfileStatus(db *gorm.DB, req UpdateProfileStatus) error {

	query := "userid = ?"

	exist := postgresql.CheckExists(db, &j, query, req.UserId)
	if !exist {
		return errors.New("Profile does not exists")
	}

	updates := map[string]any{
		"pause_notification": req.PauseNotification,
		"status_timeout":     req.StatusTimeout,
		"text":               req.Text,
		"icon":               req.Icon,
	}

	if req.ClearStatus {
		updates["text"] = ""
		updates["icon"] = ""
		updates["status_timeout"] = ""
	}

	if err := db.Model(&Profile{}).
		Where(query, req.UserId).
		Updates(updates).Error; err != nil {
		return errors.New("failed to update user profile")
	}

	if !req.ClearStatus {
		publishChannel := req.OrgId
		var user User
		if err := db.Where("id = ?", req.UserId).First(&user).Error; err != nil {
			return err
		}

		notification := Notification[ProfileStatusUpdated]
		notification.Content = ProfileStatusUpdatePayload{
			Text:       req.Text,
			Icon:       req.Icon,
			Username:   j.UserName,
			Email:      user.Email,
			ProfilePic: j.AvatarURL,
		}
		notification.ModificationDetails = &ModificationDetails{
			UserId: req.UserId,
			OrgId:  req.OrgId,
		}
		notification.NotificationId = utility.GenerateUUID()

		go centrifuge.PublishChannel(nil, publishChannel, notification)
	}

	return nil
}

func (p *Profile) GetUserByUsername(db *gorm.DB, userName string) (Profile, error) {
	var user Profile

	query := db.Where("user_name = ?", userName)

	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func (p *Profile) SetProfileImageToEmpty(db *gorm.DB, userId string) error {
	var userProfile Profile

	exists := postgresql.CheckExists(db, &userProfile, "userid = ?", userId)

	if !exists {
		return errors.New("profile does not exist")
	}

	result := db.Model(&Profile{}).Where("userid = ?", userId).Update("avatar_url", "")

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to update avatar URL")
	}

	return nil
}

func (p *Profile) GetProfileByUserId(db *gorm.DB, userId string) error {
	if err := db.Where("userid = ?", userId).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("profile not found for user ID: %s", userId)
		}
		return fmt.Errorf("failed to fetch profile: %w", err)
	}
	return nil
}

// updateProfileLinks updates the `links` text[] column for a user's profile.
// helper removed: Links are updated as part of the main update using pq.StringArray
