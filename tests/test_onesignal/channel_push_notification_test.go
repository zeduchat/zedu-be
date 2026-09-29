package test_onesignal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestChannelPushNotificationLogic(t *testing.T) {
	t.Run("PushRequest with SkipPreferenceFilter Flag", func(t *testing.T) {
		req := models.PushRequest{
			ChannelId:            "chan-123",
			OrgId:                "org-456",
			UserIds:              []string{"user-1", "user-2"},
			Message:              "Test notification message",
			ChannelName:          "general",
			UserId:               "sender-789",
			Username:             "test_sender",
			Title:                "Notification from user general",
			SkipPreferenceFilter: true,
			Payload: map[string]interface{}{
				"org_id":            "org-456",
				"channel_id":        "chan-123",
				"channel_name":      "general",
				"sender_name":       "test_sender",
				"sender_id":         "sender-789",
				"event":             "new_message",
				"notification_type": "channel",
			},
		}

		assert.True(t, req.SkipPreferenceFilter, "expected SkipPreferenceFilter to be true")
		assert.Equal(t, []string{"user-1", "user-2"}, req.UserIds)
		assert.Equal(t, "chan-123", req.ChannelId)
		assert.Equal(t, "org-456", req.OrgId)

		payloadMap, ok := req.Payload.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "new_message", payloadMap["event"])
		assert.Equal(t, "channel", payloadMap["notification_type"])
	})

	t.Run("System Message Payload Skip Check", func(t *testing.T) {
		req := models.PushRequest{
			ChannelId:            "chan-123",
			SkipPreferenceFilter: true,
			Payload: map[string]interface{}{
				"type": "system",
			},
		}

		payloadMap, ok := req.Payload.(map[string]interface{})
		assert.True(t, ok)
		msgType, exists := payloadMap["type"].(string)
		assert.True(t, exists)
		assert.Equal(t, "system", msgType)
	})

	t.Run("Empty UserIDs Handled Safely", func(t *testing.T) {
		req := models.PushRequest{
			ChannelId:            "chan-123",
			UserIds:              []string{},
			SkipPreferenceFilter: true,
		}

		assert.True(t, req.SkipPreferenceFilter)
		assert.Empty(t, req.UserIds)
	})
}
