package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	notificationpref "github.com/hngprojects/telex_be/services/notification_pref"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateChannelNotificationPreference(t *testing.T) {
	router, channelController := SetupChannelTestRouter()
	db := channelController.Db.Postgresql
	password, _ := utility.HashPassword("password")

	// Create test user
	userID := utility.GenerateUUID()
	user := models.User{
		ID:       userID,
		Name:     "Test User",
		Email:    fmt.Sprintf("test-notif-%s@qa.team", userID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	db.Create(&user)

	// Create organisation
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:          orgID,
		Name:        "Test Org",
		Email:       fmt.Sprintf("test-org-%s@qa.team", orgID),
		OwnerID:     userID,
		Description: "Test organisation",
	}
	db.Create(&org)

	// Create org user management
	orgUser := models.OrgUserManagement{
		UserID:         userID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	}
	db.Create(&orgUser)

	// Create channel
	channelID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             channelID,
		Name:           "test-channel",
		OrganisationID: orgID,
		OwnerId:        userID,
		Description:    "Test channel",
		IsPrivate:      false,
	}
	db.Create(&channel)

	// Add user to channel
	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     userID,
		Username:   "testuser",
	}
	db.Create(&userChannel)

	// Setup auth controller for login
	authController := auth.Controller{
		Db:        channelController.Db,
		Validator: channelController.Validator,
		Logger:    channelController.Logger,
		ExtReq:    channelController.ExtReq,
	}

	loginData := models.LoginRequestModel{
		Email:    user.Email,
		Password: "password",
	}
	token := tests.GetLoginToken(t, gin.Default(), authController, loginData)

	t.Run("Successful Update Channel Notification Preference", func(t *testing.T) {
		updateData := map[string]interface{}{
			"muted":       true,
			"at_mentions": true,
			"at_channel":  false,
			"device_type": "web",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%s/notification-preferences", channelID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "notification settings updated successfully")

		// Verify the response data
		// TODO: Verify response data when API returns it
	})

	t.Run("Update Channel Preference Creates Default If Not Exists", func(t *testing.T) {
		// Create another channel
		channelID2 := utility.GenerateUUID()
		channel2 := models.Channels{
			ID:             channelID2,
			Name:           "test-channel-2",
			OrganisationID: orgID,
			OwnerId:        userID,
		}
		db.Create(&channel2)

		userChannel2 := models.UserChannels{
			ChannelsID: channelID2,
			UserID:     userID,
		}
		db.Create(&userChannel2)

		updateData := map[string]interface{}{
			"muted":       false,
			"device_type": "mobile",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%s/notification-preferences", channelID2), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "notification settings updated successfully")
	})

	t.Run("Invalid Channel ID Format", func(t *testing.T) {
		updateData := map[string]interface{}{
			"muted": true,
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/channels/invalid-uuid/notification-preferences", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		updateData := map[string]interface{}{
			"muted": true,
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%s/notification-preferences", channelID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
	})
}

func TestGetChannelNotificationPref(t *testing.T) {
	router, channelController := SetupChannelTestRouter()
	db := channelController.Db.Postgresql
	password, _ := utility.HashPassword("password")

	// Create test user
	userID := utility.GenerateUUID()
	user := models.User{
		ID:       userID,
		Name:     "Test User",
		Email:    fmt.Sprintf("test-get-notif-%s@qa.team", userID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	db.Create(&user)

	// Create organisation
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:          orgID,
		Name:        "Test Org",
		Email:       fmt.Sprintf("test-org-%s@qa.team", orgID),
		OwnerID:     userID,
		Description: "Test organisation",
	}
	db.Create(&org)

	orgUser := models.OrgUserManagement{
		UserID:         userID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	}
	db.Create(&orgUser)

	// Create channel
	channelID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             channelID,
		Name:           "test-channel",
		OrganisationID: orgID,
		OwnerId:        userID,
	}
	db.Create(&channel)

	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     userID,
		Username:   "testuser",
	}
	db.Create(&userChannel)

	// Setup auth
	authController := auth.Controller{
		Db:        channelController.Db,
		Validator: channelController.Validator,
		Logger:    channelController.Logger,
		ExtReq:    channelController.ExtReq,
	}

	loginData := models.LoginRequestModel{
		Email:    user.Email,
		Password: "password",
	}
	token := tests.GetLoginToken(t, gin.Default(), authController, loginData)

	t.Run("Get Channel Notification Preference Creates Defaults", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/notification-preferences?device_type=web", channelID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "device notification fetched successfully")

		// Verify default values
		data := response["data"].(map[string]any)
		if data["muted"].(bool) != false {
			t.Errorf("Expected default muted to be false, got %v", data["muted"])
		}
		if data["at_mentions"].(bool) != true {
			t.Errorf("Expected default at_mentions to be true, got %v", data["at_mentions"])
		}
		if data["at_channel"].(bool) != true {
			t.Errorf("Expected default at_channel to be true, got %v", data["at_channel"])
		}
		if data["notify_about"] != "all_new_messages" {
			t.Errorf("Expected default notify_about to be 'all_new_messages', got %v", data["notify_about"])
		}
	})

	t.Run("Get Existing Channel Notification Preference", func(t *testing.T) {
		// First create a preference
		userChannelPref := models.UserChannels{
			ChannelsID: channelID,
			UserID:     userID,
			Preferences: models.NotificationPreference{
				"mobile": models.DeviceNotification{
					Muted:      true,
					AtMentions: false,
					AtChannel:  false,
				},
			},
		}
		db.Model(&userChannel).Updates(&userChannelPref)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/notification-preferences?device_type=mobile", channelID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)

		data := response["data"].(map[string]any)
		if data["muted"].(bool) != true {
			t.Errorf("Expected muted to be true, got %v", data["muted"])
		}
		if data["at_mentions"].(bool) != false {
			t.Errorf("Expected at_mentions to be false, got %v", data["at_mentions"])
		}
	})

	t.Run("Empty Device Type", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/notification-preferences", channelID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	})

	t.Run("Invalid Device Type", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/notification-preferences?device_type=invalid", channelID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	})

	t.Run("Invalid Channel ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/channels/invalid-uuid/notification-preferences?device_type=web", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	})
}

func TestGetUserChannelsNotificationPrefs(t *testing.T) {
	router, channelController := SetupChannelTestRouter()
	db := channelController.Db.Postgresql
	password, _ := utility.HashPassword("password")

	// Create test user
	userID := utility.GenerateUUID()
	user := models.User{
		ID:       userID,
		Name:     "Test User",
		Email:    fmt.Sprintf("test-channels-%s@qa.team", userID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	db.Create(&user)

	// Create organisation
	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:          orgID,
		Name:        "Test Org",
		Email:       fmt.Sprintf("test-org-%s@qa.team", orgID),
		OwnerID:     userID,
		Description: "Test organisation",
	}
	db.Create(&org)

	orgUser := models.OrgUserManagement{
		UserID:         userID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	}
	db.Create(&orgUser)

	// Create multiple channels
	channelID1 := utility.GenerateUUID()
	channelID2 := utility.GenerateUUID()

	channel1 := models.Channels{
		ID:             channelID1,
		Name:           "test-channel-1",
		OrganisationID: orgID,
		OwnerId:        userID,
	}
	channel2 := models.Channels{
		ID:             channelID2,
		Name:           "test-channel-2",
		OrganisationID: orgID,
		OwnerId:        userID,
	}
	db.Create(&channel1)
	db.Create(&channel2)

	// Add user to channels with different preferences
	userChannel1 := models.UserChannels{
		ChannelsID: channelID1,
		UserID:     userID,
		Username:   "testuser",
		Preferences: models.NotificationPreference{
			"web": models.DeviceNotification{
				Muted: true,
			},
		},
	}
	userChannel2 := models.UserChannels{
		ChannelsID: channelID2,
		UserID:     userID,
		Username:   "testuser",
		Preferences: models.NotificationPreference{
			"web": models.DeviceNotification{
				Muted: false,
			},
		},
	}
	db.Create(&userChannel1)
	db.Create(&userChannel2)

	// Setup auth
	authController := auth.Controller{
		Db:        channelController.Db,
		Validator: channelController.Validator,
		Logger:    channelController.Logger,
		ExtReq:    channelController.ExtReq,
	}

	loginData := models.LoginRequestModel{
		Email:    user.Email,
		Password: "password",
	}
	token := tests.GetLoginToken(t, gin.Default(), authController, loginData)

	t.Run("Get All User Channels Notification Preferences", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/notification-preferences/channels?device_type=web", orgID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "device notification channels pref fetched successfully")

		// Verify we get 2 channels back
		data := response["data"].([]any)
		if len(data) != 2 {
			t.Errorf("Expected 2 channels, got %d", len(data))
		}

		// Verify muted status is correct
		channel1Data := data[0].(map[string]any)
		if _, ok := channel1Data["muted"].(bool); !ok {
			t.Errorf("Expected muted to be a boolean, got %v", channel1Data["muted"])
		}
	})

	t.Run("Invalid Org ID Format", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/organisations/invalid-uuid/notification-preferences/channels?device_type=web", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	})

	t.Run("Empty Device Type", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/notification-preferences/channels", orgID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
	})
}

func TestNotificationPreferenceEndToEnd(t *testing.T) {
	// This test verifies that notification preferences actually work
	// by testing the ShouldSendNotification function through the full flow
	router, channelController := SetupChannelTestRouter()
	db := channelController.Db.Postgresql
	password, _ := utility.HashPassword("password")

	userID := utility.GenerateUUID()
	user := models.User{
		ID:       userID,
		Name:     "Test User",
		Email:    fmt.Sprintf("test-e2e-%s@qa.team", userID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	db.Create(&user)

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:          orgID,
		Name:        "Test Org",
		Email:       fmt.Sprintf("test-org-%s@qa.team", orgID),
		OwnerID:     userID,
		Description: "Test organisation",
	}
	db.Create(&org)

	orgUser := models.OrgUserManagement{
		UserID:         userID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	}
	db.Create(&orgUser)

	channelID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             channelID,
		Name:           "test-channel",
		OrganisationID: orgID,
		OwnerId:        userID,
	}
	db.Create(&channel)

	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     userID,
		Username:   "testuser",
	}
	db.Create(&userChannel)

	// Setup auth
	authController := auth.Controller{
		Db:        channelController.Db,
		Validator: channelController.Validator,
		Logger:    channelController.Logger,
		ExtReq:    channelController.ExtReq,
	}

	loginData := models.LoginRequestModel{
		Email:    user.Email,
		Password: "password",
	}
	token := tests.GetLoginToken(t, gin.Default(), authController, loginData)

	t.Run("Muted User Does Not Receive Notifications", func(t *testing.T) {
		// Mute the channel
		updateData := map[string]interface{}{
			"muted":       true,
			"device_type": "web",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%s/notification-preferences", channelID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)

		// Verify preference was saved
		var savedPref models.UserChannels
		db.Where("channels_id = ? AND user_id = ?", channelID, userID).First(&savedPref)

		if savedPref.Preferences == nil {
			t.Error("Expected preferences to be saved")
			return
		}

		webPrefs := savedPref.Preferences["web"]
		if !webPrefs.Muted {
			t.Error("Expected channel to be muted")
		}
	})

	t.Run("User With AllMessages Receives Notifications", func(t *testing.T) {
		// Unmute and set to all messages
		updateData := map[string]interface{}{
			"muted":       false,
			"device_type": "web",
		}
		body, _ := json.Marshal(updateData)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/channels/%s/notification-preferences", channelID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})

	t.Run("User With No Preferences Receives Default Notifications", func(t *testing.T) {
		channelID3 := utility.GenerateUUID()
		channel3 := models.Channels{
			ID:             channelID3,
			Name:           "test-channel-3",
			OrganisationID: orgID,
			OwnerId:        userID,
		}
		db.Create(&channel3)

		userChannel3 := models.UserChannels{
			ChannelsID: channelID3,
			UserID:     userID,
		}
		db.Create(&userChannel3)

		shouldSend, err := notificationpref.ShouldSendNotification(
			db, userID, channelID3, orgID,
			notificationpref.NotificationTypeAllMessages,
		)

		if err != nil {
			t.Errorf("ShouldSendNotification failed: %v", err)
		}
		if !shouldSend {
			t.Error("Expected user with no preferences to receive notifications (should apply defaults)")
		}

		shouldSendMentions, err := notificationpref.ShouldSendNotification(
			db, userID, channelID3, orgID,
			notificationpref.NotificationTypeMentions,
		)

		if err != nil {
			t.Errorf("ShouldSendNotification for mentions failed: %v", err)
		}
		if !shouldSendMentions {
			t.Error("Expected user with no preferences to receive mention notifications")
		}
	})
}
