package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type NotificationType string
type SectionType string

var (
	Updated       NotificationType = "updated"
	Deleted       NotificationType = "deleted"
	NewMessage    NotificationType = "new_message"
	ThreadSection SectionType      = "thread_message"
	ReplySection  SectionType      = "reply_message"
)

type Content struct {
	NotificationType   NotificationType       `json:"notification_type"`
	SectionType        SectionType            `json:"section"`
	ModifcationDetails ModifcationDetails     `json:"modification_ids,omitempty"`
	Content            interface{}            `json:"data,omitempty"`
	UpdateChange       map[string]interface{} `json:"update_change,omitempty"`
}

type ModifcationDetails struct {
	ThreadId  string `json:"thread_id,omitempty"`
	MessageId string `json:"message_id,omitempty"`
	ChannelId string `json:"channel_id,omitempty"`
	UserId    string `json:"user_id,omitempty"`
	OrgId     string `json:"org_id,omitempty"`
}

var Notification = map[NotificationType]Content{

	Updated: Content{
		NotificationType: Updated,
		Content:          ModifcationDetails{},
	},

	Deleted: Content{
		NotificationType: Deleted,
		Content:          ModifcationDetails{},
	},

	NewMessage: Content{
		NotificationType: NewMessage,
	},
}

// Device notifications settings

// example:

// {
// 	"mobile": {
// 	  "muted": false,
// 	  "at_mentions": true,
// 	  "at_channel": true
// 	},
// 	"web": {
// 	  "muted": true,
// 	  "at_mentions": false,
// 	  "at_channel": true
// 	}
//   }

type DeviceNotificationSettings struct {
	DeviceNotification
	ChannelsID string `json:"channels_id"`
	UserID     string `json:"user_id"`
	DeviceType string `json:"device_type" validate:"required,oneof=web mobile desktop"`
}

type DeviceNotification struct {
	Muted      bool `json:"muted"`
	AtMentions bool `json:"at_mentions"`
	AtChannel  bool `json:"at_channel"`
}

type NotificationPreference map[string]DeviceNotification

type UserChannelNotificationPref struct {
	ChannelsID  string                 `gorm:"type:uuid;primaryKey;not null" json:"channels_id"`
	UserID      string                 `gorm:"type:uuid;primaryKey;not null" json:"user_id"`
	Preferences NotificationPreference `gorm:"type:jsonb;not null;default:'{}'" json:"preferences"`
}

func (n *NotificationPreference) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &n)
}

func (n NotificationPreference) Value() (driver.Value, error) {
	return json.Marshal(n)
}

func (n *DeviceNotificationSettings) UpdateDeviceNotification(db *gorm.DB) (DeviceNotification, int, error) {
	var pref UserChannelNotificationPref
	err := db.First(&pref, "channels_id = ? AND user_id = ?", n.ChannelsID, n.UserID).Error
	if err != nil {
		return DeviceNotification{}, http.StatusBadRequest, fmt.Errorf("entry does not exist")
	}

	// Initialize if nil
	if pref.Preferences == nil {
		pref.Preferences = make(NotificationPreference)
	}

	deviceSettings := pref.Preferences[n.DeviceType]
	deviceSettings.AtChannel = n.AtChannel
	deviceSettings.AtMentions = n.AtMentions
	deviceSettings.Muted = n.Muted

	// Save back to preferences
	pref.Preferences[n.DeviceType] = deviceSettings

	return deviceSettings, http.StatusOK, db.Save(&pref).Error
}

func (n *DeviceNotificationSettings) GetOrCreateDeviceNotification(db *gorm.DB) (DeviceNotification, error) {
	var pref UserChannelNotificationPref

	exist := postgresql.CheckExists(db, pref, "channels_id = ? AND user_id = ?", n.ChannelsID, n.UserID)

	if !exist {
		// If not found, create a new preference record
		pref = UserChannelNotificationPref{
			ChannelsID:  n.ChannelsID,
			UserID:      n.UserID,
			Preferences: NotificationPreference{},
		}
	}

	// Initialize if nil
	if pref.Preferences == nil {
		pref.Preferences = make(NotificationPreference)
	}

	// Check if settings for deviceType exist
	deviceSettings, ok := pref.Preferences[n.DeviceType]
	if !ok {
		// If not exist, create default settings
		deviceSettings = DeviceNotification{
			Muted:      false,
			AtMentions: true,
			AtChannel:  true,
		}
		pref.Preferences[n.DeviceType] = deviceSettings

		// Save the updated preference
		if err := db.Save(&pref).Error; err != nil {
			return DeviceNotification{}, err
		}
	}

	return deviceSettings, nil
}

func (c *UserChannels) FetchChannelUsersNotificationPref(db *gorm.DB, logger *utility.Logger) ([]UserChannels, error) {
	var userChannel []UserChannels

	err := db.Preload("NotificationPref").Order("id").Where("channels_id = ?", c.ChannelsID).Find(userChannel).Error

	if err != nil {
		logger.Error("Error fetching:", err)
		return userChannel, err
	}

	return userChannel, nil
}

type ChannelNotificationInfo struct {
	ChannelsID  string `json:"channels_id"`
	ChannelName string `json:"channel_name"`
	Muted       bool   `json:"muted"`
}

func (n *DeviceNotificationSettings) GetUserChannelsNotificationPrefs(db *gorm.DB, ids map[string]string) ([]ChannelNotificationInfo, error) {

	var result []ChannelNotificationInfo

	err := db.
		Table("user_channel_notification_prefs AS ucnp").
		Select("ucnp.channels_id, c.name AS channel_name, COALESCE((ucnp.preferences->?->>'muted')::boolean, false) AS muted", ids["device_type"]). // You can change "web" dynamically if needed
		Joins("JOIN channels c ON c.id = ucnp.channels_id").
		Where("c.organisation_id = ? AND ucnp.user_id = ?", ids["org_id"], ids["user_id"]).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}
