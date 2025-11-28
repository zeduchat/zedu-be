package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type NotificationStyle string

type NotifyOption struct {
	Option string
}

type SoundOption struct {
	SoundName string
}

const (
	SourceDevice  PreferenceSourceType = "device"
	SourceUser    PreferenceSourceType = "user"
	SourceDefault PreferenceSourceType = "default"
)

var (
	BannerNotification   NotificationStyle = "banner"
	PopUpNotification    NotificationStyle = "pop_up"
	NotifyAllMessages                      = NotifyOption{Option: "all_new_messages"}
	NotifyDirectMentions                   = NotifyOption{Option: "direct_messages_mentions"}
	NotifyNothing                          = NotifyOption{Option: "nothing"}
)

// Direct Message Notification Setting Modes
type MessageNotification struct {
	ShowNotifications     bool        `gorm:"default:true" json:"show_notifications"`
	Sound                 SoundOption `gorm:"default:embedded" json:"sound_option"`
	ReactionNotifications bool        `gorm:"default:false" json:"reaction_notifications"`
}

// Group Notification Setting Modes
type GroupNotification struct {
	ShowNotifications     bool        `gorm:"default:true" json:"show_notifications"`
	Sound                 SoundOption `gorm:"default:embedded" json:"sound_option"`
	ReactionNotifications bool        `gorm:"default:false" json:"reaction_notifications"`
}

// InApp Notification Setting Modes
type InAppNotifications struct {
	NotificationStyle NotificationStyle `gorm:"default:true" json:"notification_type"` // banner, pop-up etc
	Sounds            SoundOption       `gorm:"default:embedded" json:"sound_option"`
	Vibrate           bool              `gorm:"default:true" json:"vibrate"`
}

// Notification Setting Modes
type NotificationSettings struct {
	MessageNotification MessageNotification `gorm:"embedded" json:"message_notification"`
	GroupNotification   GroupNotification   `gorm:"embedded" json:"group_notification"`
	Reminders           bool                `gorm:"default:true" json:""`
	InAppNotifications  InAppNotifications  `gorm:"default:embedded" json:"in_app_notifications"`
	ShowPreview         bool                `gorm:"default:false" json:"show_preview"`
}

// UserNotificationPreferences stores user-level preferences (applies to all devices)
type UserNotificationPreferences struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	NotificationSettings
}

// DeviceNotificationPreferences stores device-specific preferences (overrides user-level)
type DeviceNotificationPreferences struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"not null;index:idx_user_device" json:"user_id"`
	DeviceID  string    `gorm:"not null;index:idx_user_device" json:"device_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	NotificationSettings
}

// EffectiveNotificationPreferences stores effective notification preferences (user + device)
type EffectiveNotificationPreferences struct {
	NotificationSettings
	Source *PreferenceSource `json:"source,omitempty"`
}

// PreferenceSource tracks where each preference category came from
type PreferenceSource struct {
	MessageNotification  string `json:"message_notification"`
	GroupNotification    string `json:"group_notification"`
	ReminderNotification string `json:"reminder_notification"`
}

// PreferenceSourceType indicates the origin of a preference value
type PreferenceSourceType string

type NotificationPreferences struct {
	ID                      string         `gorm:"type:uuid;primaryKey;unique;" json:"id"`
	UserID                  uuid.UUID      `gorm:"type:uuid;" json:"user_id"`
	NotifyAbout             NotifyOption   `gorm:"embedded" json:"notify_about"`
	NotificationSchedule    bool           `gorm:"default:false" json:"notification_schedule"`
	FromHour                string         `gorm:"type:varchar(5);default:'00:00'" json:"from_hour"`
	ToHour                  string         `gorm:"type:varchar(5);default:'00:00'" json:"to_hour"`
	NotificationMethodEmail bool           `gorm:"default:false" json:"notification_method_email"`
	CreatedAt               time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`
}

func (d *NotificationPreferences) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &d)

	if err != nil {
		return err
	}

	return nil
}

func (d *NotificationPreferences) Update(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &d)
	return err
}

func (d *NotificationPreferences) GetUserDataByID(db *gorm.DB, userID string) (NotificationPreferences, error) {
	var user NotificationPreferences

	query := db.Where("user_id::uuid = ?", userID)
	if err := query.First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

func DefaultNotificationPreferences() NotificationSettings {
	return NotificationSettings{
		MessageNotification: MessageNotification{
			ShowNotifications: true,
			Sound: SoundOption{
				SoundName: "Note",
			},
			ReactionNotifications: false,
		},
		GroupNotification: GroupNotification{
			ShowNotifications: true,
			Sound: SoundOption{
				SoundName: "Note",
			},
			ReactionNotifications: false,
		},
		Reminders: true,
		InAppNotifications: InAppNotifications{
			NotificationStyle: BannerNotification,
			Sounds: SoundOption{
				SoundName: "Note",
			},
			Vibrate: true,
		},
		ShowPreview: true,
	}
}

// UpdateNotificationPreferencesRequest for partial updates
type UpdateNotificationPreferencesRequest struct {
	MessageNotification  *MessageNotificationUpdate `json:"message_notification,omitempty"`
	GroupNotification    *GroupNotificationUpdate   `json:"group_notification,omitempty"`
	ReminderNotification *bool                      `json:"reminder_notification,omitempty"`
}

type MessageNotificationUpdate struct {
	ShowNotifications     *bool               `json:"show_notifications,omitempty"`
	Sound                 *SoundOption        `json:"sound_option,omitempty"`
	ReactionNotifications *bool               `json:"reaction_notifications,omitempty"`
	InAppNotifications    *InAppNotifications `json:"in_app_notifications,omitempty"`
	ShowPreview           *bool               `json:"show_preview,omitempty"`
}

type GroupNotificationUpdate struct {
	ShowNotifications     *bool        `json:"show_notifications,omitempty"`
	Sound                 *SoundOption `json:"sound_option,omitempty"`
	ReactionNotifications *bool        `json:"reaction_notifications,omitempty"`
}

// MergePreferences creates effective preferences by merging device, user, and default preferences
func MergePreferences(device *DeviceNotificationPreferences, user *UserNotificationPreferences) EffectiveNotificationPreferences {
	defaults := DefaultNotificationPreferences()
	effective := EffectiveNotificationPreferences{
		NotificationSettings: defaults,
		Source: &PreferenceSource{
			MessageNotification:  string(SourceDefault),
			GroupNotification:    string(SourceDefault),
			ReminderNotification: string(SourceDefault),
		},
	}

	// Merge MessageNotification
	if user != nil {
		effective.MessageNotification = user.MessageNotification
		effective.Source.MessageNotification = string(SourceUser)
	}
	if device != nil {
		effective.MessageNotification = device.MessageNotification
		effective.Source.MessageNotification = string(SourceDevice)
	}

	// Merge GroupNotification
	if user != nil {
		effective.GroupNotification = user.GroupNotification
		effective.Source.GroupNotification = string(SourceUser)
	}
	if device != nil {
		effective.GroupNotification = device.GroupNotification
		effective.Source.GroupNotification = string(SourceDevice)
	}

	// Merge ReminderNotification
	if user != nil {
		effective.Reminders = user.Reminders
		effective.Source.ReminderNotification = string(SourceUser)
	}
	if device != nil {
		effective.Reminders = device.Reminders
		effective.Source.ReminderNotification = string(SourceDevice)
	}

	return effective
}
