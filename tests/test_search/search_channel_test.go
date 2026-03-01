package test_search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelSearchEndpoint(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("channeluser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "ChannelUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("channeluser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("channeluser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "ChannelUser",
		LastName:    "Two",
		Password:    "password",
		UserName:    fmt.Sprintf("channeluser2_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	loginData2 := models.LoginRequestModel{
		Email:    user2SignUpData.Email,
		Password: user2SignUpData.Password,
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	authRouter := gin.Default()
	tst.SignupUser(t, authRouter, authController, user1SignUpData, false)

	authRouter2 := gin.Default()
	tst.SignupUser(t, authRouter2, authController, user2SignUpData, false)

	loginRouter := gin.Default()
	token1 := tst.GetLoginToken(t, loginRouter, authController, loginData1)

	loginRouter2 := gin.Default()
	token2 := tst.GetLoginToken(t, loginRouter2, authController, loginData2)

	var user1, user2 models.User
	db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1)
	db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2)

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	var role models.OrgRole
	if err := db.Postgresql.First(&role).Error; err != nil {
		t.Fatalf("Failed to fetch any OrgRole: %v", err)
	}

	// Make user2 join org
	userOrg := models.OrgUserManagement{
		UserID:         user2.ID,
		OrganisationID: org.ID,
		RoleID:         role.ID,
		Status:         "active",
	}
	db.Postgresql.Create(&userOrg)

	r := SetupSearchRouter(db, logger)

	privateChannel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           "private_channel_test",
		OrganisationID: org.ID,
		IsPrivate:      true,
		OwnerId:        user1.ID,
	}
	db.Postgresql.Create(&privateChannel)

	userChannel := models.UserChannels{
		ChannelsID: privateChannel.ID,
		UserID:     user1.ID,
	}
	db.Postgresql.Create(&userChannel)

	threadId := utility.GenerateUUID()
	thread := map[string]any{
		"thread_id":   threadId,
		"channels_id": privateChannel.ID,
		"user_id":     user1.ID,
		"org_id":      privateChannel.OrganisationID,
		"message":     "Secret message in isolated channel search",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	}
	elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadId, thread, logger)

	time.Sleep(2 * time.Second)

	t.Run("Authorized member can search private channel", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/channel/%s/?query=Secret", privateChannel.ID), nil)

		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if dataInt, ok := resp["data"]; ok && dataInt != nil {
			data := dataInt.([]interface{})
			if len(data) != 1 {
				t.Errorf("Member should see 1 result, got %d", len(data))
			}

			// Verify that the result actually belongs to the specified channel
			resultMap := data[0].(map[string]interface{})
			channelData, ok := resultMap["channel"].(map[string]interface{})
			if !ok || channelData["channel_id"] != privateChannel.ID {
				t.Errorf("Result did not come from the searched channel. Expected %s, got %v", privateChannel.ID, resultMap)
			}
		} else {
			t.Errorf("Member should see 1 result but got null/none. Response: %v", resp)
		}
	})

	t.Run("Unauthorized user cannot search private channel", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/channel/%s/?query=Secret", privateChannel.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden && rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 403 Forbidden for non-member, got %d", rr.Code)
		}
	})

	// Add a normal (public) search case
	publicChannel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           "public_channel_test",
		OrganisationID: privateChannel.OrganisationID,
		IsPrivate:      false,
		OwnerId:        user1.ID,
	}
	db.Postgresql.Create(&publicChannel)

	userChannelPublic := models.UserChannels{
		ChannelsID: publicChannel.ID,
		UserID:     user1.ID,
	}
	db.Postgresql.Create(&userChannelPublic)

	threadIdPublic := utility.GenerateUUID()
	threadPublic := map[string]any{
		"thread_id":   threadIdPublic,
		"channels_id": publicChannel.ID,
		"user_id":     user1.ID,
		"org_id":      publicChannel.OrganisationID,
		"message":     "Public message in isolated channel search",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	}
	elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadIdPublic, threadPublic, logger)

	time.Sleep(2 * time.Second)

	t.Run("Authorized member can search public channel", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/search/channel/%s/?query=Public", publicChannel.ID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if dataInt, ok := resp["data"]; ok && dataInt != nil {
			data := dataInt.([]interface{})
			if len(data) != 1 {
				t.Errorf("Member should see 1 result, got %d", len(data))
			}

			// Verify that the result actually belongs to the specified public channel
			resultMap := data[0].(map[string]interface{})
			channelData, ok := resultMap["channel"].(map[string]interface{})
			if !ok || channelData["channel_id"] != publicChannel.ID {
				t.Errorf("Result did not come from the searched public channel. Expected %s, got %v", publicChannel.ID, resultMap)
			}
		} else {
			t.Errorf("Member should see 1 result but got null/none. Response: %v", resp)
		}
	})
}
