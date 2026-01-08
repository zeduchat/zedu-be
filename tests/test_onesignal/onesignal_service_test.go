package test_onesignal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/onesignal"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
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

	err := onesignal.OptionalSendNotification(logger, "test-sub-id", "Test Title", "Test Body")

	assert.NoError(t, err)
}

func TestOptionalSendBatchNotifications(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	err := onesignal.OptionalSendBatchNotifications(logger, []string{"sub-1", "sub-2"}, "Test Title", "Test Body")

	assert.NoError(t, err)
}

func TestSendNotificationNoClient(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	err := onesignal.SendNotification(logger, "test-sub-id", "Test Title", "Test Body")

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

	err := onesignal.SendBatchNotifications(logger, []string{}, "Test Title", "Test Body")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no subscription IDs provided")
}
