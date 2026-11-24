package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUsersInChannelSearch(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()
	validatorRef := validator.New()

	currUUID := uuid.Must(uuid.NewV4()).String()

	authCtrl := auth.Controller{
		Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true},
	}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()

	authUrl := r.Group("/api/v1/auth")
	{
		authUrl.POST("/register", authCtrl.RegisterUser)
		authUrl.POST("/login", authCtrl.LoginUser)
	}

	channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("", channelCtrl.CreateChannel)
		channelUrl.GET("/:channelId/users", channelCtrl.GetUsersInChannel)
	}

	ownerSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("searchowner_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "SearchOwner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("searchowner_%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, ownerSignUpData, false)
	ownerToken := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    ownerSignUpData.Email,
		Password: ownerSignUpData.Password,
	})

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("saviour_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Saviour",
		LastName:    "UniquePerson",
		Password:    "password",
		UserName:    fmt.Sprintf("saviour_%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, user1SignUpData, false)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("OrgSearchTest%s", currUUID),
		Description: "Org description",
		Email:       ownerSignUpData.Email,
		Type:        "test",
		Location:    "test",
		Country:     "test",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, ownerToken)

	var user1Model models.User
	db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1Model)
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user1Model.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	createChanData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("chan_search_%s", currUUID),
		Username:       fmt.Sprintf("chan_user_%s", currUUID),
		OrganisationID: orgId,
		Description:    "Test channel for search",
	}
	channelID, _ := tst.CreateChannels(t, r, channelCtrl, db, createChanData, ownerToken)

	db.Postgresql.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     user1Model.ID,
		Username:   user1SignUpData.UserName,
	})

	t.Run("Search for matching user returns matching user", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/users?page=1&limit=20&search=saviour", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		rrSearch := httptest.NewRecorder()
		r.ServeHTTP(rrSearch, req)
		tst.AssertStatusCode(t, rrSearch.Code, http.StatusOK)

		resp := tst.ParseResponse(rrSearch)
		usersData, ok := resp["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be array of users")
		}

		if len(usersData) != 1 {
			t.Errorf("Expected 1 matching user for 'saviour', got %d", len(usersData))
		}
	})

	t.Run("Search for non-matching query returns 0 users", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/users?page=1&limit=20&search=nonexistentuser12345", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		rrSearch := httptest.NewRecorder()
		r.ServeHTTP(rrSearch, req)
		tst.AssertStatusCode(t, rrSearch.Code, http.StatusOK)

		resp := tst.ParseResponse(rrSearch)
		usersData, ok := resp["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be array of users")
		}

		if len(usersData) != 0 {
			t.Errorf("Expected 0 matching users, got %d", len(usersData))
		}
	})

	t.Run("Deactivated user excluded when exclude_deactivated=true", func(t *testing.T) {
		db.Postgresql.Model(&models.OrgUserManagement{}).
			Where("user_id = ? AND organisation_id = ?", user1Model.ID, orgId).
			Update("is_deactivated", true)

		fetch := func(query string) []interface{} {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/channels/%s/users?page=1&limit=20&search=saviour%s", channelID, query), nil)
			req.Header.Set("Authorization", "Bearer "+ownerToken)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			tst.AssertStatusCode(t, rr.Code, http.StatusOK)
			usersData, _ := tst.ParseResponse(rr)["data"].([]interface{})
			return usersData
		}

		if users := fetch(""); len(users) != 1 {
			t.Errorf("Expected 1 user without filter, got %d", len(users))
		}
		if users := fetch("&exclude_deactivated=true"); len(users) != 0 {
			t.Errorf("Expected 0 users with exclude_deactivated, got %d", len(users))
		}

		active, err := models.FilterActiveUserIDs(db.Postgresql, orgId, []string{user1Model.ID})
		if err != nil {
			t.Fatalf("FilterActiveUserIDs failed: %v", err)
		}
		if len(active) != 0 {
			t.Errorf("Expected deactivated user to be filtered out, got %v", active)
		}
	})
}
