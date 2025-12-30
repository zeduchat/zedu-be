package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestRemoveMultipleMembersFromChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validate := validator.New()
	db := storage.Connection()

	authController := auth.Controller{
		Db:        db,
		Validator: validate,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}
	channelController := channel.Controller{Db: db, Validator: validate, Logger: logger}
	orgController := organisation.Controller{Db: db, Validator: validate, Logger: logger}

	r := gin.Default()
	channelGroup := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelGroup.POST("/remove-multiple", channelController.RemoveMultipleMembersFromChannel)
	}

	// Owner user
	ownerEmail := fmt.Sprintf("owner_%s@qa.team", utility.GenerateUUID())
	ownerSignup := models.CreateUserRequestModel{Email: ownerEmail, Password: "password"}
	ownerLogin := models.LoginRequestModel{Email: ownerSignup.Email, Password: ownerSignup.Password}
	tst.SignupUser(t, r, authController, ownerSignup, false)
	ownerToken := tst.GetLoginToken(t, r, authController, ownerLogin)
	if ownerToken == "" {
		t.Fatalf("failed to obtain owner token")
	}

	var owner models.User
	if err := db.Postgresql.Where("email = ?", ownerSignup.Email).First(&owner).Error; err != nil {
		t.Fatalf("failed to fetch owner user: %v", err)
	}

	// Secondary user to remove
	memberEmail := fmt.Sprintf("member_%s@qa.team", utility.GenerateUUID())
	memberSignup := models.CreateUserRequestModel{Email: memberEmail, Password: "password"}
	memberLogin := models.LoginRequestModel{Email: memberSignup.Email, Password: memberSignup.Password}
	tst.SignupUser(t, gin.Default(), authController, memberSignup, false)
	tst.GetLoginToken(t, gin.Default(), authController, memberLogin) // ensure user exists

	var member models.User
	if err := db.Postgresql.Where("email = ?", memberSignup.Email).First(&member).Error; err != nil {
		t.Fatalf("failed to fetch member user: %v", err)
	}

	// Create organisation and channel
	orgReq := models.CreateOrgRequestModel{
		Name:        "RemoveTestOrg",
		Description: "org for remove test",
		Email:       ownerEmail,
		Type:        "type1",
		Location:    "lagos",
		Country:     "ng",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgController, orgReq, ownerToken)

	channelReq := models.CreateChannelsRequest{
		Name:           "remove-channel",
		Username:       "remove-username",
		OrganisationID: orgID,
		Description:    "channel for remove",
	}
	channelID, _ := tst.CreateChannels(t, r, channelController, db, channelReq, ownerToken)
	if channelID == "" {
		t.Fatalf("failed to create channel")
	}

	// Add member to channel directly
	if err := db.Postgresql.Create(&models.UserChannels{ChannelsID: channelID, UserID: member.ID}).Error; err != nil {
		t.Fatalf("failed to add member to channel setup: %v", err)
	}

	body := models.RemoveMultipleMembersRequest{
		ChannelID: channelID,
		UserIDs:   []string{member.ID},
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("failed to encode body: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/channels/remove-multiple", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ownerToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	var check models.UserChannels
	err := db.Postgresql.Where("channels_id = ? AND user_id = ?", channelID, member.ID).First(&check).Error
	if err == nil {
		t.Fatalf("expected member to be removed from channel")
	}
}
