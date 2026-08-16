package notificationpref

import (
	"fmt"
	"strings"
	"time"

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
	return ShouldSendNotificationWithTime(db, userID, channelID, orgID, notifType, time.Now())
}

func ShouldSendNotificationWithTime(db *gorm.DB, userID, channelID, orgID string, notifType NotificationType, nowUTC time.Time) (bool, error) {
	userLoc := time.UTC

	if db != nil && userID != "" {
		var profModel models.Profile
		prof, err := profModel.GetOrCreateProfileForOrg(db, userID, orgID)
		if err == nil {
			if prof.PauseNotification {
				return false, nil
			}
			if prof.Timezone != "" {
				if loc, err := time.LoadLocation(prof.Timezone); err == nil {
					userLoc = loc
				}
			}
		}
	}

	if db != nil {
		prefs, err := GetEffectivePreferences(db, userID, channelID, orgID)
		if err == nil && len(prefs) > 0 {
			for _, devicePrefs := range prefs {
				if shouldSendForDevice(devicePrefs, notifType, userLoc, nowUTC) {
					return true, nil
				}
			}
			return false, nil
		}
	}

	defaultPrefs := models.GetDefaultChannelDeviceNotification()
	return shouldSendForDevice(defaultPrefs, notifType, userLoc, nowUTC), nil
}

func shouldSendForDevice(devicePrefs models.DeviceNotification, notifType NotificationType, userLoc *time.Location, nowUTC time.Time) bool {
	if devicePrefs.Muted {
		return false
	}

	if devicePrefs.TimeRange != "" && !IsTimeWithinRange(devicePrefs.TimeRange, userLoc, nowUTC) {
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

func IsTimeWithinRange(timeRangeStr string, loc *time.Location, nowUTC time.Time) bool {
	if timeRangeStr == "" {
		return true
	}

	parts := strings.Split(timeRangeStr, "-")
	if len(parts) != 2 {
		return true
	}

	startMin, err1 := parseTimeOfDay(parts[0])
	endMin, err2 := parseTimeOfDay(parts[1])
	if err1 != nil || err2 != nil {
		return true
	}

	if loc == nil {
		loc = time.UTC
	}

	userLocalTime := nowUTC.In(loc)
	currentMin := userLocalTime.Hour()*60 + userLocalTime.Minute()

	if startMin <= endMin {
		return currentMin >= startMin && currentMin <= endMin
	} else {
		return currentMin >= startMin || currentMin <= endMin
	}
}

func parseTimeOfDay(timeStr string) (int, error) {
	timeStr = strings.TrimSpace(timeStr)
	t, err := time.Parse("3:04 PM", timeStr)
	if err != nil {
		t, err = time.Parse("15:04", timeStr)
		if err != nil {
			return 0, err
		}
	}
	return t.Hour()*60 + t.Minute(), nil
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

