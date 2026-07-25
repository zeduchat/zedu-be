package test_onesignal

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/onesignal"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	onesignalService "github.com/hngprojects/telex_be/services/onesignal"
	push_notifications "github.com/hngprojects/telex_be/services/pushNotifications"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestConnectOneSignal(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	tests := []struct {
		Name   string
		Config config.OneSignal
		Expect bool
	}{
		{
			Name: "Successful initialization",
			Config: config.OneSignal{
				AppID:      "test-app-id",
				RestAPIKey: "test-api-key",
				Enabled:    true,
			},
			Expect: true,
		},
		{
			Name: "Disabled in config",
			Config: config.OneSignal{
				AppID:      "test-app-id",
				RestAPIKey: "test-api-key",
				Enabled:    false,
			},
			Expect: false,
		},
		{
			Name: "Empty AppID",
			Config: config.OneSignal{
				AppID:      "",
				RestAPIKey: "test-api-key",
				Enabled:    true,
			},
			Expect: false,
		},
		{
			Name: "Empty RestAPIKey",
			Config: config.OneSignal{
				AppID:      "test-app-id",
				RestAPIKey: "",
				Enabled:    true,
			},
			Expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			onesignal.Client.AppID = ""
			onesignal.Client.ApiKey = ""

			onesignal.ConnectOneSignal(logger, tt.Config)

			if tt.Expect {
				assert.NotEmpty(t, onesignal.Client.AppID)
				assert.Equal(t, tt.Config.AppID, onesignal.Client.AppID)
				assert.NotEmpty(t, onesignal.Client.ApiKey)
				assert.Equal(t, tt.Config.RestAPIKey, onesignal.Client.ApiKey)
			} else {
				assert.Empty(t, onesignal.Client.AppID)
				assert.Empty(t, onesignal.Client.ApiKey)
			}
		})
	}
}

func TestOptionalSendNotification(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	req := models.PushRequest{
		Title:   "Test Title",
		Message: "Test Body",
	}

	err := onesignal.OptionalSendNotification(logger, "test-sub-id", req, db.Postgresql, "")
	assert.NoError(t, err)
}

func TestOptionalSendBatchNotifications(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	req := models.PushRequest{
		Title:   "Test Title",
		Message: "Test Body",
	}

	err := onesignal.OptionalSendBatchNotifications(logger, []string{"sub-1", "sub-2"}, req, db.Postgresql, []string{})
	assert.NoError(t, err)
}

func TestSendNotificationNoClient(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	req := models.PushRequest{
		Title:   "Test Title",
		Message: "Test Body",
	}

	err := onesignal.SendNotification(logger, "test-sub-id", req, db.Postgresql, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OneSignal client not initialized")
}

func TestSendBatchNotificationsEmpty(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	onesignal.Client.AppID = "test-app"
	onesignal.Client.ApiKey = "test-key"

	req := models.PushRequest{
		Title:   "Test Title",
		Message: "Test Body",
	}

	err := onesignal.SendBatchNotifications(logger, []string{}, req, db.Postgresql, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subscription IDs provided")
}

func TestGetNotificationsByUserAndOrgService(t *testing.T) {
	db := storage.Connection()
	tst.Setup()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	testUser := CreateTestUser(t, db)
	senderUser := CreateTestUser(t, db)
	orgID := utility.GenerateUUID()

	if err := db.Postgresql.AutoMigrate(&models.OneSignalNotification{}); err != nil {
		t.Fatalf("Failed to auto migrate OneSignalNotification: %v", err)
	}
	db.Postgresql.Exec("DELETE FROM onesignal_notifications")

	notif := models.OneSignalNotification{
		ID:                      utility.GenerateUUID(),
		UserID:                  testUser.ID,
		OrgID:                   &orgID,
		OneSignalNotificationID: "notif-service-123",
		Title:                   "Service Test Notification",
		Message:                 "Test Message for Service",
		Payload:                 map[string]interface{}{"sender_id": senderUser.ID, "key": "val"},
		Status:                  models.OneSignalNotificationStatusPending,
		SentAt:                  time.Now(),
	}
	require.NoError(t, db.Postgresql.Create(&notif).Error)

	notifs, pag, err := onesignalService.GetNotificationsByUserAndOrg(db.Postgresql, testUser.ID, orgID, 1, 10)
	require.NoError(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, int64(1), pag.TotalItems)

	// Verify default_avatar_url is added to payload
	expectedAvatarURL := avatar.GenerateDefaultAvatarURL(senderUser.ID)
	require.NotNil(t, notifs[0].Payload)
	assert.Equal(t, expectedAvatarURL, notifs[0].Payload["default_avatar_url"])
}

func TestResolvePushTitleAndBody(t *testing.T) {
	db := storage.Connection()

	randomUser := "user_" + utility.RandomString(8)
	randomChan := "chan_" + utility.RandomString(8)

	// Standard channel title
	req1 := models.PushRequest{
		ChannelName: randomChan,
	}
	assert.Equal(t, "#"+randomChan, push_notifications.ResolvePushTitle(req1, db.Postgresql))

	// DM channel title (no # prefix)
	req2 := models.PushRequest{
		ChannelName: randomUser,
		Payload: map[string]interface{}{
			"notification_type": "dm",
		},
	}
	assert.Equal(t, randomUser, push_notifications.ResolvePushTitle(req2, db.Postgresql))

	// Empty channel name fallback to sender username
	req3 := models.PushRequest{
		ChannelName: "",
		Title:       "# ",
		Username:    randomUser,
	}
	assert.Equal(t, randomUser, push_notifications.ResolvePushTitle(req3, db.Postgresql))

	// Empty channel name fallback to New Message when no username or title
	req4 := models.PushRequest{
		ChannelName: "",
		Title:       "# ",
	}
	assert.Equal(t, "New Message", push_notifications.ResolvePushTitle(req4, db.Postgresql))

	// Body formatting with username prefix
	msgContent := "hello " + utility.RandomString(6)
	expectedFormattedBody := fmt.Sprintf("(%s): %s", randomUser, msgContent)
	assert.Equal(t, expectedFormattedBody, push_notifications.ResolvePushBody(randomUser, msgContent))
	// Body formatting when already prefixed
	assert.Equal(t, expectedFormattedBody, push_notifications.ResolvePushBody(randomUser, expectedFormattedBody))
}
