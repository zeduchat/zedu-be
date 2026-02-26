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
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	channelController "github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserChannelsSearch(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	chanCtrl := channelController.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	// Create test user
	userSignupData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user_%s@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%d", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("user_%s", currUUID),
	}

	signupRouter := gin.Default()
	tst.SignupUser(t, signupRouter, authCtrl, userSignupData, false)

	loginData := models.LoginRequestModel{
		Email:    userSignupData.Email,
		Password: userSignupData.Password,
	}
	loginRouter := gin.Default()
	token := tst.GetLoginToken(t, loginRouter, authCtrl, loginData)
	userID := tst.GetUserIDFromToken(t, token, db)

	// Create organization
	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    fmt.Sprintf("ChanSearchOrg_%s", currUUID),
		Email:   userSignupData.Email,
		OwnerID: userID,
	}
	db.Postgresql.Create(&org)

	// Create 2 channels
	channels := []models.Channels{
		{
			ID:             utility.GenerateUUID(),
			Name:           "target-channel-unique",
			Description:    "Target Channel",
			OrganisationID: org.ID,
			OwnerId:        userID,
			CreatedAt:      time.Now(),
		},
		{
			ID:             utility.GenerateUUID(),
			Name:           "other-channel",
			Description:    "Other Channel",
			OrganisationID: org.ID,
			OwnerId:        userID,
			CreatedAt:      time.Now(),
		},
	}

	for _, ch := range channels {
		db.Postgresql.Create(&ch)
		// Add user to channel
		db.Postgresql.Create(&models.UserChannels{
			ChannelsID: ch.ID,
			UserID:     userID,
			CreatedAt:  time.Now(),
		})
	}

	r := gin.Default()
	r.GET("/api/v1/organisations/:org_id/channels", middleware.Authorize(db.Postgresql), chanCtrl.GetUserChannels)

	tests := []struct {
		name          string
		search        string
		expectedCount int
		expectedFound bool
	}{
		{
			name:          "Search by exact name",
			search:        "target-channel-unique",
			expectedCount: 1,
			expectedFound: true,
		},
		{
			name:          "Search by partial name",
			search:        "unique",
			expectedCount: 1,
			expectedFound: true,
		},
		{
			name:          "Search no match",
			search:        "nonexistent",
			expectedCount: 0,
			expectedFound: false,
		},
		{
			name:          "Empty search (returns all)",
			search:        "",
			expectedCount: 2,
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/v1/organisations/%s/channels?search=%s", org.ID, tt.search)
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)

			var result map[string]interface{}
			json.Unmarshal(resp.Body.Bytes(), &result)

			data, _ := result["data"].([]interface{})
			assert.Equal(t, tt.expectedCount, len(data))

			if tt.expectedFound && tt.search != "" {
				found := false
				for _, d := range data {
					ch := d.(map[string]interface{})
					if ch["name"] == "target-channel-unique" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find target channel in results")
			}
		})
	}
}

func TestGetUserChannelsSorting(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	chanCtrl := channelController.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	// Create test user
	userSignupData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("sort_user_%s@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%d", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		Password:    "password",
		UserName:    fmt.Sprintf("sort_user_%s", currUUID),
	}

	signupRouter := gin.Default()
	tst.SignupUser(t, signupRouter, authCtrl, userSignupData, false)

	loginData := models.LoginRequestModel{
		Email:    userSignupData.Email,
		Password: userSignupData.Password,
	}
	loginRouter := gin.Default()
	token := tst.GetLoginToken(t, loginRouter, authCtrl, loginData)
	userID := tst.GetUserIDFromToken(t, token, db)

	// Create organization
	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    fmt.Sprintf("ChanSortOrg_%s", currUUID),
		Email:   userSignupData.Email,
		OwnerID: userID,
	}
	db.Postgresql.Create(&org)

	// Create 3 channels with different created_at and we'll mock/influence unread count via the DB
	// Note: In real app, unread count comes from ElasticSearch.
	// To test the SORTING logic itself, we either need a true integration test with Elastic
	// or we accept that we're verifying the API returns results.
	// However, we can verify that the order of results matches our expectation if we can influence it.

	channelNames := []string{"Channel A", "Channel B", "Channel C"}
	channelIDs := make([]string, 3)

	for i, name := range channelNames {
		channelIDs[i] = utility.GenerateUUID()
		ch := models.Channels{
			ID:             channelIDs[i],
			Name:           name,
			OrganisationID: org.ID,
			OwnerId:        userID,
			CreatedAt:      time.Now().Add(time.Duration(i) * time.Hour), // A < B < C in time
		}
		db.Postgresql.Create(&ch)
		db.Postgresql.Create(&models.UserChannels{
			ChannelsID: ch.ID,
			UserID:     userID,
			CreatedAt:  time.Now(),
		})
	}

	r := gin.Default()
	r.GET("/api/v1/organisations/:org_id/channels", middleware.Authorize(db.Postgresql), chanCtrl.GetUserChannels)

	t.Run("Verify Default Sort Order (Time-based)", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/organisations/%s/channels", org.ID)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var result map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &result)
		data, _ := result["data"].([]interface{})

		// Expected order: Channel C, Channel B, Channel A (descending by CreateAt)
		assert.True(t, len(data) >= 3)
		assert.Equal(t, "Channel C", data[0].(map[string]interface{})["name"])
		assert.Equal(t, "Channel B", data[1].(map[string]interface{})["name"])
		assert.Equal(t, "Channel A", data[2].(map[string]interface{})["name"])
	})
}
