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
	dmController "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetDmChannelsSearch(t *testing.T) {
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

	dmCtrl := dmController.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
	}

	// Create 2 test users: searching user and target user
	users := make([]struct {
		SignupData models.CreateUserRequestModel
		Token      string
		UserID     string
	}, 2)

	for i := 0; i < 2; i++ {
		users[i].SignupData = models.CreateUserRequestModel{
			Email:       fmt.Sprintf("user%d_%s@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%d", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			Password:    "password",
			UserName:    fmt.Sprintf("searching_user%d_%s", i, currUUID),
		}
		if i == 1 {
			users[i].SignupData.UserName = "target_username_unique"
			users[i].SignupData.FirstName = "TargetFirstName"
			users[i].SignupData.LastName = "TargetLastName"
		}

		signupRouter := gin.Default()
		tst.SignupUser(t, signupRouter, authCtrl, users[i].SignupData, false)

		loginData := models.LoginRequestModel{
			Email:    users[i].SignupData.Email,
			Password: users[i].SignupData.Password,
		}
		loginRouter := gin.Default()
		users[i].Token = tst.GetLoginToken(t, loginRouter, authCtrl, loginData)
		users[i].UserID = tst.GetUserIDFromToken(t, users[i].Token, db)
	}

	// Create organization
	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    fmt.Sprintf("SearchTestOrg_%s", currUUID),
		Email:   users[0].SignupData.Email,
		OwnerID: users[0].UserID,
	}
	db.Postgresql.Create(&org)

	// Create DM channel between user 0 and user 1
	dmChannelID := utility.GenerateUUID()
	dmChannel := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        users[0].UserID,
		ChannelId:     dmChannelID,
		OrgId:         org.ID,
		ParticipantId: &users[1].UserID,
		ChannelType:   "dm",
		ChatType:      "user",
		CreatedAt:     time.Now(),
	}
	db.Postgresql.Create(&dmChannel)

	// Create another user that shouldn't match
	otherUserEmail := fmt.Sprintf("other_%s@qa.team", currUUID)
	otherUserSignup := models.CreateUserRequestModel{
		Email:    otherUserEmail,
		Password: "password",
		UserName: "other_user",
	}
	tst.SignupUser(t, gin.Default(), authCtrl, otherUserSignup, false)

	var otherUser models.User
	db.Postgresql.Where("email = ?", otherUserEmail).First(&otherUser)

	// Create DM between user 0 and other user
	dmChannelOther := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        users[0].UserID,
		ChannelId:     utility.GenerateUUID(),
		OrgId:         org.ID,
		ParticipantId: &otherUser.ID,
		ChannelType:   "dm",
		ChatType:      "user",
		CreatedAt:     time.Now(),
	}
	db.Postgresql.Create(&dmChannelOther)

	r := gin.Default()
	r.GET("/api/v1/dm/:org_id", middleware.Authorize(db.Postgresql), dmCtrl.GetDmChannels)

	tests := []struct {
		name          string
		search        string
		expectedCount int
		expectedFound bool
	}{
		{
			name:          "Search by username",
			search:        "target_username",
			expectedCount: 1,
			expectedFound: true,
		},
		{
			name:          "Search by email part",
			search:        "user1",
			expectedCount: 1,
			expectedFound: true,
		},
		{
			name:          "Search no match",
			search:        "nonexistentpattern",
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
			url := fmt.Sprintf("/api/v1/dm/%s?search=%s", org.ID, tt.search)
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			req.Header.Set("Authorization", "Bearer "+users[0].Token)

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
					dm := d.(map[string]interface{})
					if dm["participant_id"] == users[1].UserID {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find target user in results")
			}
		})
	}
}
