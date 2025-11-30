package models

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type NotifyOption struct {
	Option string
}

// --- Enums and Constants ---

// PreferenceSourceType indicates the origin of a preference value
type PreferenceSourceType string
type NotificationStyle string

const (
	SourceDevice  PreferenceSourceType = "device"
	SourceUser    PreferenceSourceType = "user"
	SourceDefault PreferenceSourceType = "default"

	BannerNotification NotificationStyle = "banner"
	PopUpNotification  NotificationStyle = "pop_up"
)

var (
	NotifyAllMessages    = NotifyOption{Option: "all_new_messages"}
	NotifyDirectMentions = NotifyOption{Option: "direct_messages_mentions"}
	NotifyNothing        = NotifyOption{Option: "nothing"}
)

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

type SoundOption struct {
	SoundName string
}

// Direct Message Notification Setting Modes
type MessageNotification struct {
	ShowNotifications     bool        `gorm:"default:true" json:"show_notifications"`
	Sound                 SoundOption `gorm:"embedded" json:"sound_option"`
	ReactionNotifications bool        `gorm:"default:false" json:"reaction_notifications"`
}

// Group Notification Setting Modes
type GroupNotification struct {
	ShowNotifications     bool        `gorm:"default:true" json:"show_notifications"`
	Sound                 SoundOption `gorm:"embedded" json:"sound_option"`
	ReactionNotifications bool        `gorm:"default:false" json:"reaction_notifications"`
}

// InApp Notification Setting Modes
type InAppNotifications struct {
	NotificationStyle NotificationStyle `gorm:"default:true" json:"notification_type"` // banner, pop-up etc
	Sounds            SoundOption       `gorm:"embedded" json:"sound_option"`
	Vibrate           bool              `gorm:"default:true" json:"vibrate"`
}

// Notification Preferences Modes
type NotificationSettings struct {
	MessageNotification MessageNotification `gorm:"embedded" json:"message_notification"`
	GroupNotification   GroupNotification   `gorm:"embedded" json:"group_notification"`
	Reminders           bool                `gorm:"default:true" json:"reminders"`
	InAppNotifications  InAppNotifications  `gorm:"embedded" json:"in_app_notifications"`
	ShowPreview         bool                `gorm:"default:false" json:"show_preview"`
}

// Notification Preferences Request Modes
type NotificationSettingsUpdateRequest struct {
	MessageNotification *MessageNotification `json:"message_notification,omitempty"`
	GroupNotification   *GroupNotification   `json:"group_notification,omitempty"`
	Reminders           *bool                `json:"reminders,omitempty"`
	InAppNotifications  *InAppNotifications  `json:"in_app_notifications,omitempty"`
	ShowPreview         *bool                `json:"show_preview,omitempty"`
}

// UserNotificationPreferences stores user-level preferences (applies to all devices)
type UserNotificationSetting struct {
	ID        string    `gorm:"type:uuid;primaryKey;unique;" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;" json:"user_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	NotificationSettings
}

// DeviceNotificationPreferences stores device-specific preferences (overrides user-level)
type DeviceNotificationSetting struct {
	ID        string    `gorm:"type:uuid;primaryKey;unique;" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;" json:"user_id"`
	DeviceID  string    `gorm:"not null;index:idx_user_device" json:"device_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	NotificationSettings
}

// EffectiveNotificationSetting stores effective notification settings (user + device)
type EffectiveNotificationSetting struct {
	DeviceID string `json:"device_id"`
	NotificationSettings
	Source *PreferenceSource `json:"source,omitempty"`
}

// UpdateNotificationPreferencesRequest for partial updates
type UpdateNotificationPreferencesRequest struct {
	MessageNotification  *MessageNotificationUpdate `json:"message_notification,omitempty"`
	GroupNotification    *GroupNotificationUpdate   `json:"group_notification,omitempty"`
	ReminderNotification *bool                      `json:"reminder_notification,omitempty"`
	InAppNotifications   *InAppNotificationsUpdate  `json:"in_app_notifications"`
	ShowPreview          *bool                      `json:"show_preview"`
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

type InAppNotificationsUpdate struct {
	NotificationStyle *NotificationStyle `gorm:"default:true" json:"notification_type"` // banner, pop-up etc
	Sounds            *SoundOption       `gorm:"default:embedded" json:"sound_option"`
	Vibrate           *bool              `gorm:"default:true" json:"vibrate"`
}

// PreferenceSource tracks where each preference category came from
type PreferenceSource struct {
	MessageNotification  string `json:"message_notification"`
	GroupNotification    string `json:"group_notification"`
	ReminderNotification string `json:"reminder_notification"`
	InAppNotifications   string `json:"in_app_notifications"`
	ShowPreview          string `json:"show_preview"`
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

// MergePreferences creates effective preferences by merging device, user, and default preferences
func MergePreferences(device *DeviceNotificationSetting, user *UserNotificationSetting) EffectiveNotificationSetting {
	defaults := DefaultNotificationPreferences()
	effective := EffectiveNotificationSetting{
		NotificationSettings: defaults,
		Source: &PreferenceSource{
			MessageNotification:  string(SourceDefault),
			GroupNotification:    string(SourceDefault),
			ReminderNotification: string(SourceDefault),
		},
	}

	// Merge MessageNotification by Device
	if device != nil {
		effective.MessageNotification = device.MessageNotification
		effective.Source.MessageNotification = string(SourceDevice)
	}
	if user != nil {
		effective.MessageNotification = user.MessageNotification
		effective.Source.MessageNotification = string(SourceUser)
	}

	// Merge GroupNotification By Device
	if device != nil {
		effective.GroupNotification = device.GroupNotification
		effective.Source.GroupNotification = string(SourceDevice)
	}
	if user != nil {
		effective.GroupNotification = user.GroupNotification
		effective.Source.GroupNotification = string(SourceUser)
	}

	// Merge ReminderNotification by User
	if user != nil {
		effective.Reminders = user.Reminders
		effective.Source.ReminderNotification = string(SourceUser)
	}
	if device != nil {
		effective.Reminders = device.Reminders
		effective.Source.ReminderNotification = string(SourceDevice)
	}

	// Merge InAppNotifications by Device
	if device != nil {
		effective.InAppNotifications = device.InAppNotifications
	}
	if user != nil {
		effective.InAppNotifications = user.InAppNotifications
		effective.Source.InAppNotifications = string(SourceUser)
	}

	// Merge ShowPreview by Device
	if device != nil {
		effective.ShowPreview = device.ShowPreview
	}
	if user != nil {
		effective.ShowPreview = user.ShowPreview
		effective.Source.ShowPreview = string(SourceUser)
	}

	return effective
}

func (d *UserNotificationSetting) CreateUserNotificationSetting(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &d)

	if err != nil {
		return err
	}

	return nil
}

func (d *UserNotificationSetting) UpdateUserNotificationSetting(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &d)
	return err
}

func (d *UserNotificationSetting) GetUserNotificationSettingByID(db *gorm.DB, userID string) (UserNotificationSetting, error) {
	var userSetting UserNotificationSetting

	query := db.Where("user_id::uuid = ?", userID)
	if err := query.First(&userSetting).Error; err != nil {
		return userSetting, err
	}

	return userSetting, nil
}

func (d *DeviceNotificationSetting) CreateDeviceNotificationSetting(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &d)

	if err != nil {
		return err
	}

	return nil
}

func (d *DeviceNotificationSetting) UpdateDeviceNotificationSetting(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &d)
	return err
}

func (d *DeviceNotificationSetting) GetDeviceNotificationSettingByID(db *gorm.DB, userID string, deviceID string) (DeviceNotificationSetting, error) {
	var deviceSetting DeviceNotificationSetting
	query := db.Where("user_id = ? AND device_id = ?", userID, deviceID)
	if err := query.First(&deviceSetting).Error; err != nil {
		return deviceSetting, err
	}

	return deviceSetting, nil
}
