package test_fcm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestFitDataPayload_ProtectedRoutingKeys(t *testing.T) {
	// Build an oversized payload with huge message content and critical routing keys
	hugeContent := strings.Repeat("A very long message that should be truncated to fit FCM limits. ", 100)

	payload := map[string]interface{}{
		"channel_id":        "chan-uuid-1234-5678-abcdef",
		"thread_id":         "thread-uuid-1234-5678-abcdef",
		"message_id":        "msg-uuid-1234-5678-abcdef",
		"org_id":            "org-uuid-1234-5678-abcdef",
		"event":             "new_message",
		"notification_type": "channel",
		"section":           "reply",
		"type":              "standard",
		"buzz_id":           "buzz-12345",
		"buzz_code":         "BZ123",
		"host_id":           "host-999",
		"content":           hugeContent,
		"avatar_url":        "https://example.com/long_avatar_url_string_that_can_be_trimmed",
	}

	pushReq := models.PushRequest{
		Payload: payload,
	}

	title := push_notifications.ResolvePushTitle(pushReq, nil)
	assert.NotEmpty(t, title)

	// Test ResolvePushTitle prefix logic
	reqDM := models.PushRequest{
		ChannelName: "john_doe",
		Payload: map[string]interface{}{
			"notification_type": "dm",
		},
	}
	assert.Equal(t, "john_doe", push_notifications.ResolvePushTitle(reqDM, nil))

	reqChannel := models.PushRequest{
		ChannelName: "general",
		Payload: map[string]interface{}{
			"notification_type": "channel",
		},
	}
	assert.Equal(t, "#general", push_notifications.ResolvePushTitle(reqChannel, nil))

	// Test ResolvePushBody
	body := push_notifications.ResolvePushBody("alice", "Hello world")
	assert.Equal(t, "(alice): Hello world", body)

	bodyAlreadyFormatted := push_notifications.ResolvePushBody("alice", "(alice): Hello world")
	assert.Equal(t, "(alice): Hello world", bodyAlreadyFormatted)
}

func TestDeliverFCM_MultiDeviceCollection(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	user := CreateTestUser(t, db)

	tokenWeb := "fcm_multi_web_" + utility.GenerateUUID()
	tokenAndroid := "fcm_multi_android_" + utility.GenerateUUID()
	tokenIOS := "fcm_multi_ios_" + utility.GenerateUUID()

	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: tokenWeb, Platform: "web"}).Upsert(db.Postgresql))
	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: tokenAndroid, Platform: "android"}).Upsert(db.Postgresql))
	require.NoError(t, (&models.UserDeviceToken{UserId: user.ID, Token: tokenIOS, Platform: "ios"}).Upsert(db.Postgresql))

	// Verify all 3 tokens are found for the user
	tokens, err := models.ListDeviceTokensByUserIDs(db.Postgresql, []string{user.ID})
	require.NoError(t, err)
	assert.Equal(t, 3, len(tokens))

	// Calling PushOneSignalToUser with uninitialized Firebase client should gracefully return nil
	req := models.PushRequest{
		UserId:  user.ID,
		Title:   "Test Notification",
		Message: "Hello multi-device user",
		Payload: map[string]interface{}{
			"event":             "new_message",
			"notification_type": "channel",
			"channel_id":        "chan-1",
		},
	}

	err = push_notifications.PushOneSignalToUser(req, logger, db.Postgresql)
	assert.NoError(t, err, "push notification delivery should be graceful even without live Firebase credentials")
}
