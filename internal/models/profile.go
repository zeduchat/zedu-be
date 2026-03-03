package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
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
	RiverJobID        *int64         `gorm:"type:bigint;index" json:"river_job_id,omitempty"`
	WorkspaceID       string         `gorm:"type:varchar(255)" json:"workspace_id"`
	Track             string         `gorm:"type:varchar(255)" json:"track"`
	Links             pq.StringArray `gorm:"type:text[]" json:"links"`
	Online            bool           `gorm:"type:boolean;default:true" json:"online"`
	IsActive          bool           `gorm:"type:boolean;default:true" json:"is_active"`
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
	DefaultAvatarURL  string   `json:"default_avatar_url"`
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
	Online            bool     `json:"online"`
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
	StatusExpiry      string `json:"status_expiry"` // Natural language expiry like "30 minutes"
	ClearStatus       bool   `json:"clear_status"`
	Online            bool   `json:"online"`
	StatusVisibility  string `json:"status_visibility"` // Stored in DB but not used in new API
	UserId            string
	OrgId             string
}

// PartialStatusUpdate holds optional fields for status patch requests.
// Pointer fields let us detect whether a field was supplied so we can
// avoid overwriting existing values.
type PartialStatusUpdate struct {
	Text   *string `json:"text"`
	Emoji  *string `json:"emoji"`
	Expiry *string `json:"expiry"`
	UserID string  `json:"-"`
}

// SetStatusRequest holds fields for setting a new status via POST.
// Text is required; emoji and expiry are optional.
type SetStatusRequest struct {
	Text   string  `json:"text" validate:"required,min=1,max=255"`
	Emoji  *string `json:"emoji,omitempty" validate:"omitempty,max=64,no_whitespace,emoji"`
	Expiry *string `json:"expiry,omitempty" validate:"omitempty,status_expiry"`
	UserID string  `json:"-"`
}

type UpdateUserPresenceRequest struct {
	IsActive bool   `json:"online" validate:"boolean"`
	UserID   string `json:"-"`
	OrgID    string `json:"-"`
}

// UserStatus represents the persisted status for responses.
type UserStatus struct {
	Text       string `json:"text"`
	Emoji      string `json:"emoji"`
	Expiry     int64  `json:"expiry"`
	Visibility string `json:"visibility"`
	Online     bool   `json:"online"`
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

func (j *Profile) UpdateProfileStatus(db *gorm.DB, req UpdateProfileStatus, logger *utility.Logger) (UserStatus, int, error) {
	query := "userid = ?"

	exist := postgresql.CheckExists(db, &j, query, req.UserId)
	if !exist {
		return UserStatus{}, http.StatusNotFound, errors.New("profile does not exist")
	}

	// Load current profile to get timezone and existing river_job_id
	if err := db.Where("userid = ?", req.UserId).First(&j).Error; err != nil {
		return UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to load profile: %w", err)
	}

	// Parse status expiry if provided
	var expiryTimestamp int64
	if req.StatusExpiry != "" {
		var err error
		expiryTimestamp, err = j.ParseStatusExpiry(req.StatusExpiry)
		if err != nil {
			return UserStatus{}, http.StatusBadRequest, fmt.Errorf("invalid expiry: %w", err)
		}
	}

	// Build updates map
	updates := map[string]any{
		"pause_notification": req.PauseNotification,
		"online":             req.Online,
	}

	// Handle text/icon - only update if provided
	if req.Text != "" || req.Icon != "" {
		updates["text"] = req.Text
		updates["icon"] = req.Icon
	}

	// Handle clear status
	if req.ClearStatus {
		updates["text"] = ""
		updates["icon"] = ""
		updates["status_timeout"] = ""
		ctx := context.Background()
		if j.RiverJobID != nil && storage.DB.River != nil {
			_, cancelErr := storage.DB.River.JobCancel(ctx, *j.RiverJobID)
			if cancelErr != nil {
				logger.Error("failed to cancel clear status job %d: %v", *j.RiverJobID, cancelErr)
			} else {
				logger.Info("Cancelled clear status job %d", *j.RiverJobID)
			}
		}
		updates["river_job_id"] = nil
	} else {
		// Handle status timeout/expiry
		if req.StatusExpiry != "" {
			if expiryTimestamp > 0 {
				updates["status_timeout"] = strconv.FormatInt(expiryTimestamp, 10)
			} else {
				updates["status_timeout"] = ""
			}
		} else if req.StatusTimeout != "" {
			// Use pre-formatted timeout if provided
			updates["status_timeout"] = req.StatusTimeout
		}

		// Clear river_job_id if no expiry provided (cancels any existing auto-clear job)
		if req.StatusExpiry == "" && req.StatusTimeout == "" {
			ctx := context.Background()
			if j.RiverJobID != nil && storage.DB.River != nil {
				_, cancelErr := storage.DB.River.JobCancel(ctx, *j.RiverJobID)
				if cancelErr != nil {
					logger.Error("failed to cancel clear status job %d: %v", *j.RiverJobID, cancelErr)
				} else {
					logger.Info("Cancelled clear status job %d", *j.RiverJobID)
				}
			}
			updates["river_job_id"] = nil
		}
	}

	// Apply updates
	if err := db.Model(&Profile{}).
		Where(query, req.UserId).
		Updates(updates).Error; err != nil {
		return UserStatus{}, http.StatusInternalServerError, errors.New("failed to update user profile")
	}

	// Reload profile to get updated values
	if err := db.Where("userid = ?", req.UserId).First(&j).Error; err != nil {
		return UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to reload profile: %w", err)
	}

	// Build response status
	expiry := int64(0)
	if j.StatusTimeout != "" {
		if parsed, err := strconv.ParseInt(j.StatusTimeout, 10, 64); err == nil {
			expiry = parsed
		}
	}

	visibility := "public"
	if j.StatusVisibility != "" {
		visibility = j.StatusVisibility
	}

	status := UserStatus{
		Text:       j.Text,
		Emoji:      j.Icon,
		Expiry:     expiry,
		Visibility: visibility,
		Online:     j.Online,
	}

	// Publish notification if not clearing
	if !req.ClearStatus && logger != nil {
		logger.Info("status updated", "user_id", req.UserId)
		notification := Notification[StatusUpdate]
		notification.SectionType = ChannelsSection
		notification.NotificationId = utility.GenerateUUID()
		notification.ModificationDetails = &ModificationDetails{
			UserId: req.UserId,
		}
		notification.Content = struct {
			UserID string     `json:"user_id"`
			Status UserStatus `json:"status"`
		}{
			UserID: req.UserId,
			Status: status,
		}

		channelID := fmt.Sprintf("user:%s", req.UserId)
		if err := centrifuge.PublishChannel(logger, channelID, notification); err != nil {
			logger.Error("failed to publish status update event", "error", err, "channel_id", channelID)
		}
	}

	return status, http.StatusOK, nil
}

// ParseStatusExpiry converts a natural language expiry string to a Unix timestamp.
func (p *Profile) ParseStatusExpiry(expiryStr string) (int64, error) {
	if expiryStr == "" {
		return 0, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(expiryStr))
	now := time.Now()

	var loc *time.Location
	if p.Timezone != "" {
		var err error
		loc, err = time.LoadLocation(p.Timezone)
		if err != nil {
			loc = time.UTC
		}
	} else {
		loc = time.UTC
	}

	nowInTz := now.In(loc)

	switch normalized {
	case "30 minutes", "30 minute":
		return nowInTz.Add(30 * time.Minute).Unix(), nil

	case "1 hour", "1 hr":
		return nowInTz.Add(1 * time.Hour).Unix(), nil

	case "today":
		endOfDay := time.Date(
			nowInTz.Year(),
			nowInTz.Month(),
			nowInTz.Day(),
			23, 59, 59, 0,
			loc,
		)
		return endOfDay.Unix(), nil

	case "this week":
		daysUntilSunday := (7 - int(nowInTz.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		endOfWeek := time.Date(
			nowInTz.Year(),
			nowInTz.Month(),
			nowInTz.Day(),
			23, 59, 59, 0,
			loc,
		).AddDate(0, 0, daysUntilSunday)
		return endOfWeek.Unix(), nil

	case "don't remove", "dont remove", "do not remove":
		return 0, nil

	default:
		return 0, fmt.Errorf("invalid expiry value: %s", expiryStr)
	}
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
