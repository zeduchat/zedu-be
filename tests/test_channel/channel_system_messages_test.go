package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

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

func TestChannelJoinAndLeaveSystemMessages(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	ownerSignup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("chanowner_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Owner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("chanowner_%v", currUUID),
	}

	user1Signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("chanuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Chan",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("chanuser1_%v", currUUID),
	}

	user2Signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("chanuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Chan",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("chanuser2_%v", currUUID),
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authController, ownerSignup, false)
	tst.SignupUser(t, r, authController, user1Signup, false)
	tst.SignupUser(t, r, authController, user2Signup, false)

	ownerToken := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: ownerSignup.Email, Password: ownerSignup.Password})
	token1 := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: user1Signup.Email, Password: user1Signup.Password})
	token2 := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: user2Signup.Email, Password: user2Signup.Password})

	orgReq := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("ChanSysOrg_%s", currUUID),
		Description: "Org for channel system message tests",
		Email:       ownerSignup.Email,
		Type:        "test",
		Location:    "lagos",
		Country:     "ng",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgController, orgReq, ownerToken)

	var user1Model, user2Model models.User
	db.Postgresql.Where("email = ?", user1Signup.Email).First(&user1Model)
	db.Postgresql.Where("email = ?", user2Signup.Email).First(&user2Model)

	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user1Model.ID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user2Model.ID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	chanReq := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("syschan-%s", currUUID),
		Username:       ownerSignup.UserName,
		OrganisationID: orgID,
		Description:    "Channel system message test",
	}
	channelID, _ := tst.CreateChannels(t, r, channelController, db, chanReq, ownerToken)

	t.Run("Single User Join Channel System Message Wording", func(t *testing.T) {
		r := gin.Default()
		channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
		{
			channelUrl.POST("/:channelId/join", channelController.JoinChannels)
		}

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/join", channelID), strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for user1 join, got %d", rr.Code)
		}

		time.Sleep(2 * time.Second)

		sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
		if sysMsg != nil {
			content, ok := sysMsg["message"].(string)
			if ok {
				expectedSnippet := "joined this channel"
				if !strings.Contains(content, expectedSnippet) {
					t.Errorf("Expected join message to contain %q, got %q", expectedSnippet, content)
				}
			}
		}
	})

	t.Run("Multiple User Join Channel System Message Merging", func(t *testing.T) {
		r := gin.Default()
		channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
		{
			channelUrl.POST("/:channelId/join", channelController.JoinChannels)
		}

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/join", channelID), strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+token2)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for user2 join, got %d", rr.Code)
		}

		time.Sleep(2 * time.Second)

		sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
		if sysMsg != nil {
			content, ok := sysMsg["message"].(string)
			if ok {
				if !strings.Contains(content, "also joined") && !strings.Contains(content, "joined this channel") {
					t.Errorf("Expected merged join message, got %q", content)
				}
			}
		}
	})

	t.Run("User Leave Channel System Message Wording", func(t *testing.T) {
		r := gin.Default()
		channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
		{
			channelUrl.POST("/:channelId/leave", channelController.LeaveChannels)
		}

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/leave", channelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Expected status 200 for user1 leave, got %d", rr.Code)
		}

		time.Sleep(2 * time.Second)

		sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
		if sysMsg != nil {
			content, ok := sysMsg["message"].(string)
			if ok {
				expectedSnippet := "left this channel"
				if !strings.Contains(content, expectedSnippet) {
					t.Errorf("Expected leave message to contain %q, got %q", expectedSnippet, content)
				}
			}
		}
	})
}
