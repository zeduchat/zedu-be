package notificationpref

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

type NotificationType string

const (
	NotificationTypeAllMessages   NotificationType = "all_messages"
	NotificationTypeMentions      NotificationType = "mentions"
	NotificationTypeAtChannel     NotificationType = "at_channel"
	NotificationTypeDirectMessage NotificationType = "direct_message"
	NotificationTypeFileShare     NotificationType = "file_share"
	NotificationTypeBuzz          NotificationType = "buzz"
	NotificationTypeThreadReply   NotificationType = "thread_reply"
	NotificationTypeMention       NotificationType = "mention"
)

func ShouldSendNotification(db *gorm.DB, userID, channelID, orgID string, notifType NotificationType) (bool, error) {
	prefs, err := GetEffectivePreferences(db, userID, channelID, orgID)
	if err != nil {
		return false, err
	}

	if len(prefs) == 0 {
		defaultPrefs := models.GetDefaultChannelDeviceNotification()
		return shouldSendForDevice(defaultPrefs, notifType), nil
	}

	for _, devicePrefs := range prefs {
		if shouldSendForDevice(devicePrefs, notifType) {
			return true, nil
		}
	}

	return false, nil
}

func shouldSendForDevice(devicePrefs models.DeviceNotification, notifType NotificationType) bool {
	if devicePrefs.Muted {
		return false
	}

	switch notifType {
	case NotificationTypeAllMessages:
		return devicePrefs.NotifyAbout == models.AllMessages
	case NotificationTypeMentions, NotificationTypeMention:
		return devicePrefs.AtMentions || devicePrefs.NotifyAbout == models.AllMessages
	case NotificationTypeAtChannel:
		return devicePrefs.AtChannel || devicePrefs.NotifyAbout == models.AllMessages
	case NotificationTypeDirectMessage:
		return devicePrefs.NotifyAbout != models.Nothing
	case NotificationTypeFileShare, NotificationTypeBuzz, NotificationTypeThreadReply:
		return devicePrefs.NotifyAbout != models.Nothing
	default:
		return devicePrefs.NotifyAbout == models.AllMessages
	}
}

func FilterUsersByPreferences(db *gorm.DB, userIDs []string, channelID, orgID string, notifType NotificationType) ([]string, error) {
	filtered := make([]string, 0, len(userIDs))

	for _, userID := range userIDs {
		shouldSend, err := ShouldSendNotification(db, userID, channelID, orgID, notifType)
		if err != nil {
			return nil, fmt.Errorf("error checking preferences for user %s: %w", userID, err)
		}
		if shouldSend {
			filtered = append(filtered, userID)
		}
	}

	return filtered, nil
}

func GetEffectivePreferences(db *gorm.DB, userID, channelID, orgID string) (models.NotificationPreference, error) {
	var userChannel models.UserChannels
	err := db.Where("user_id = ? AND channels_id = ?", userID, channelID).
		First(&userChannel).Error

	if err == nil && len(userChannel.Preferences) > 0 {
		return userChannel.Preferences, nil
	}

	var orgUser models.OrgUserManagement
	err = db.Where("user_id = ? AND organisation_id = ?", userID, orgID).
		First(&orgUser).Error

	if err == nil && len(orgUser.Preferences) > 0 {
		return orgUser.Preferences, nil
	}

	return models.NotificationPreference{}, nil
}
