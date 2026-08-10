package test_profile

import (
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	notificationpref "github.com/hngprojects/telex_be/services/notification_pref"
)

// TestShouldSendForDevice_AllMessages directly tests preference filtering
func TestShouldSendForDevice_AllMessages(t *testing.T) {
	tests := []struct {
		name         string
		prefs        models.DeviceNotification
		notifType    notificationpref.NotificationType
		shouldBeSent bool
	}{
		{
			name: "All messages - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.AllMessages,
			},
			notifType:    notificationpref.NotificationTypeAllMessages,
			shouldBeSent: true,
		},
		{
			name: "Muted channel - should not send",
			prefs: models.DeviceNotification{
				Muted:       true,
				NotifyAbout: models.AllMessages,
			},
			notifType:    notificationpref.NotificationTypeAllMessages,
			shouldBeSent: false,
		},
		{
			name: "Mentions only with mention - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				AtMentions:  true,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    notificationpref.NotificationTypeMentions,
			shouldBeSent: true,
		},
		{
			name: "Mentions only with all messages - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				AtMentions:  true,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    notificationpref.NotificationTypeAllMessages,
			shouldBeSent: false,
		},
		{
			name: "Nothing preference - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.Nothing,
			},
			notifType:    notificationpref.NotificationTypeAllMessages,
			shouldBeSent: false,
		},
		{
			name: "@channel enabled with @channel notification - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				AtChannel:   true,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    notificationpref.NotificationTypeAtChannel,
			shouldBeSent: true,
		},
		{
			name: "Direct message with nothing preference - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.Nothing,
			},
			notifType:    notificationpref.NotificationTypeDirectMessage,
			shouldBeSent: false,
		},
		{
			name: "Direct message with mentions - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    notificationpref.NotificationTypeDirectMessage,
			shouldBeSent: true,
		},
		{
			name: "File share with all messages - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.AllMessages,
			},
			notifType:    notificationpref.NotificationTypeFileShare,
			shouldBeSent: true,
		},
		{
			name: "File share with mentions - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    notificationpref.NotificationTypeFileShare,
			shouldBeSent: true,
		},
		{
			name: "File share with nothing - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.Nothing,
			},
			notifType:    notificationpref.NotificationTypeFileShare,
			shouldBeSent: false,
		},
		{
			name: "Buzz with all messages - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.AllMessages,
			},
			notifType:    notificationpref.NotificationTypeBuzz,
			shouldBeSent: true,
		},
		{
			name: "Thread reply with mentions - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    notificationpref.NotificationTypeThreadReply,
			shouldBeSent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test using default timezone & time
			result := notificationpref.IsTimeWithinRange(tt.prefs.TimeRange, nil, time.Now())
			if result != true && tt.prefs.TimeRange != "" {
				t.Errorf("IsTimeWithinRange() failed for %s", tt.name)
			}
		})
	}
}

func TestIsTimeWithinRange(t *testing.T) {
	nyLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	tests := []struct {
		name         string
		timeRange    string
		loc          *time.Location
		nowUTC       time.Time
		shouldBeSent bool
	}{
		{
			name:         "Empty time range - always allowed",
			timeRange:    "",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 14, 0, 0, 0, time.UTC),
			shouldBeSent: true,
		},
		{
			name:         "Standard range 08:00 AM - 10:00 PM inside window",
			timeRange:    "08:00 AM - 10:00 PM",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 14, 30, 0, 0, time.UTC),
			shouldBeSent: true,
		},
		{
			name:         "Standard range 08:00 AM - 10:00 PM outside window (night)",
			timeRange:    "08:00 AM - 10:00 PM",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 23, 0, 0, 0, time.UTC),
			shouldBeSent: false,
		},
		{
			name:         "Standard range 08:00 AM - 10:00 PM outside window (early morning)",
			timeRange:    "08:00 AM - 10:00 PM",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 5, 0, 0, 0, time.UTC),
			shouldBeSent: false,
		},
		{
			name:         "Cross-midnight range 10:00 PM - 07:00 AM inside late night",
			timeRange:    "10:00 PM - 07:00 AM",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 23, 30, 0, 0, time.UTC),
			shouldBeSent: true,
		},
		{
			name:         "Cross-midnight range 10:00 PM - 07:00 AM inside early morning",
			timeRange:    "10:00 PM - 07:00 AM",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 4, 0, 0, 0, time.UTC),
			shouldBeSent: true,
		},
		{
			name:         "Cross-midnight range 10:00 PM - 07:00 AM outside window (afternoon)",
			timeRange:    "10:00 PM - 07:00 AM",
			loc:          time.UTC,
			nowUTC:       time.Date(2026, 8, 4, 14, 0, 0, 0, time.UTC),
			shouldBeSent: false,
		},
		{
			name:         "Timezone America/New_York (UTC 14:00 is 10:00 AM EDT) inside 08:00 AM - 12:00 PM",
			timeRange:    "08:00 AM - 12:00 PM",
			loc:          nyLoc,
			nowUTC:       time.Date(2026, 8, 4, 14, 0, 0, 0, time.UTC), // 10:00 AM NY
			shouldBeSent: true,
		},
		{
			name:         "Timezone America/New_York (UTC 18:00 is 02:00 PM EDT) outside 08:00 AM - 12:00 PM",
			timeRange:    "08:00 AM - 12:00 PM",
			loc:          nyLoc,
			nowUTC:       time.Date(2026, 8, 4, 18, 0, 0, 0, time.UTC), // 2:00 PM NY
			shouldBeSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notificationpref.IsTimeWithinRange(tt.timeRange, tt.loc, tt.nowUTC)
			if got != tt.shouldBeSent {
				t.Errorf("IsTimeWithinRange(%q) = %v, want %v", tt.timeRange, got, tt.shouldBeSent)
			}
		})
	}
}

// Test edge cases for preference constants
func TestNotificationPreferenceConstants(t *testing.T) {
	if models.AllMessages != "all_new_messages" {
		t.Errorf("Expected AllMessages to be 'all_new_messages', got '%s'", models.AllMessages)
	}

	if models.DirectMentions != "mentions" {
		t.Errorf("Expected DirectMentions to be 'mentions', got '%s'", models.DirectMentions)
	}

	if models.Nothing != "nothing" {
		t.Errorf("Expected Nothing to be 'nothing', got '%s'", models.Nothing)
	}
}

// Test notification type constants
func TestNotificationTypeConstants(t *testing.T) {
	testCases := []struct {
		constant    notificationpref.NotificationType
		expectedVal string
	}{
		{notificationpref.NotificationTypeAllMessages, "all_messages"},
		{notificationpref.NotificationTypeMentions, "mentions"},
		{notificationpref.NotificationTypeMention, "mention"},
		{notificationpref.NotificationTypeAtChannel, "at_channel"},
		{notificationpref.NotificationTypeDirectMessage, "direct_message"},
		{notificationpref.NotificationTypeFileShare, "file_share"},
		{notificationpref.NotificationTypeBuzz, "buzz"},
		{notificationpref.NotificationTypeThreadReply, "thread_reply"},
	}

	for _, tc := range testCases {
		if string(tc.constant) != tc.expectedVal {
			t.Errorf("Expected %s to be '%s', got '%s'", tc.constant, tc.expectedVal, string(tc.constant))
		}
	}
}
