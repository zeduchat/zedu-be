package notificationpref

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

// Test shouldSendForDevice function directly (unit test without DB)
func TestShouldSendForDevice_AllMessages(t *testing.T) {
	tests := []struct {
		name         string
		prefs        models.DeviceNotification
		notifType    NotificationType
		shouldBeSent bool
	}{
		{
			name: "All messages - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.AllMessages,
			},
			notifType:    NotificationTypeAllMessages,
			shouldBeSent: true,
		},
		{
			name: "Muted channel - should not send",
			prefs: models.DeviceNotification{
				Muted:       true,
				NotifyAbout: models.AllMessages,
			},
			notifType:    NotificationTypeAllMessages,
			shouldBeSent: false,
		},
		{
			name: "Mentions only with mention - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				AtMentions:  true,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    NotificationTypeMentions,
			shouldBeSent: true,
		},
		{
			name: "Mentions only with all messages - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				AtMentions:  true,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    NotificationTypeAllMessages,
			shouldBeSent: false,
		},
		{
			name: "Nothing preference - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.Nothing,
			},
			notifType:    NotificationTypeAllMessages,
			shouldBeSent: false,
		},
		{
			name: "@channel enabled with @channel notification - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				AtChannel:   true,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    NotificationTypeAtChannel,
			shouldBeSent: true,
		},
		{
			name: "Direct message with nothing preference - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.Nothing,
			},
			notifType:    NotificationTypeDirectMessage,
			shouldBeSent: false,
		},
		{
			name: "Direct message with mentions - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    NotificationTypeDirectMessage,
			shouldBeSent: true,
		},
		{
			name: "File share with all messages - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.AllMessages,
			},
			notifType:    NotificationTypeFileShare,
			shouldBeSent: true,
		},
		{
			name: "File share with mentions - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    NotificationTypeFileShare,
			shouldBeSent: true,
		},
		{
			name: "File share with nothing - should not send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.Nothing,
			},
			notifType:    NotificationTypeFileShare,
			shouldBeSent: false,
		},
		{
			name: "Buzz with all messages - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.AllMessages,
			},
			notifType:    NotificationTypeBuzz,
			shouldBeSent: true,
		},
		{
			name: "Thread reply with mentions - should send",
			prefs: models.DeviceNotification{
				Muted:       false,
				NotifyAbout: models.DirectMentions,
			},
			notifType:    NotificationTypeThreadReply,
			shouldBeSent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSendForDevice(tt.prefs, tt.notifType)
			if result != tt.shouldBeSent {
				t.Errorf("shouldSendForDevice() = %v, want %v", result, tt.shouldBeSent)
			}
		})
	}
}

// Test edge cases for preference constants
func TestNotificationPreferenceConstants(t *testing.T) {
	// Verify that our constants match the expected values
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
	// Verify notification type constants are set correctly
	testCases := []struct {
		constant    NotificationType
		expectedVal string
	}{
		{NotificationTypeAllMessages, "all_messages"},
		{NotificationTypeMentions, "mentions"},
		{NotificationTypeMention, "mention"},
		{NotificationTypeAtChannel, "at_channel"},
		{NotificationTypeDirectMessage, "direct_message"},
		{NotificationTypeFileShare, "file_share"},
		{NotificationTypeBuzz, "buzz"},
		{NotificationTypeThreadReply, "thread_reply"},
	}

	for _, tc := range testCases {
		if string(tc.constant) != tc.expectedVal {
			t.Errorf("Expected %s to be '%s', got '%s'", tc.constant, tc.expectedVal, string(tc.constant))
		}
	}
}
