package test_integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gosimple/slug"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/agents"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetCustomAgentAppsPreview(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authCtrl, loginData)
	userID := tst.GetUserIDFromToken(t, token, db)

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Some Random description",
		Email:       userSignUpData.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	agentNames := []string{
		fmt.Sprintf("Agent Two %s", currUUID),
		fmt.Sprintf("Agent One %s", currUUID),
		fmt.Sprintf("Agent Three %s", currUUID),
	}
	var integrationIDs []string
	for _, name := range agentNames {
		intID := utility.GenerateUUID()
		integrationIDs = append(integrationIDs, intID)

		agent := models.Integrations{
			ID:             intID,
			Name:           name,
			AppDescription: "Test Agent Description",
			AppLogo:        "https://example.com/logo.png",
			IsSystem:       true,
			OwnerID:        userID,
			PreSharedKey:   utility.GenerateUUID(),
		}
		if err := db.Postgresql.Create(&agent).Error; err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		orgInt := models.OrganisationIntegrations{
			ID:            utility.GenerateUUID(),
			OrgID:         orgID,
			IntegrationID: intID,
			IsActive:      true,
			AppName:       name,
			AppLogo:       "https://example.com/logo.png",
			OwnerID:       userID,
		}
		if err := db.Postgresql.Create(&orgInt).Error; err != nil {
			t.Fatalf("Failed to link agent to org: %v", err)
		}

		chanID := utility.GenerateUUID()
		dmChan := models.DmChannels{
			ID:            utility.GenerateUUID(),
			ChannelId:     chanID,
			OrgId:         orgID,
			UserId:        userID,
			ParticipantId: &intID,
			ChannelType:   "dm",
			ChatType:      "bot",
		}
		if err := db.Postgresql.Create(&dmChan).Error; err != nil {
			t.Fatalf("Failed to create DM channel: %v", err)
		}

		if strings.HasPrefix(name, "Agent Two") {
			threadDoc := models.ThreadDocument{
				ID:         utility.GenerateUUID(),
				ChannelsID: chanID,
				Content:    "Hello from Agent Two",
				UserId:     userID,
				CreatedAt:  time.Now().Add(-1 * time.Hour),
				Type:       "thread",
			}
			if err := threadDoc.CreateThread(db, logger); err != nil {
				t.Logf("Warning: Failed to create thread in Elastic: %v", err)
			}
		}

		if strings.HasPrefix(name, "Agent One") {
			threadDoc := models.ThreadDocument{
				ID:         utility.GenerateUUID(),
				ChannelsID: chanID,
				Content:    "Hello from Agent One",
				UserId:     userID,
				CreatedAt:  time.Now(),
				Type:       "thread",
			}
			if err := threadDoc.CreateThread(db, logger); err != nil {
				t.Logf("Warning: Failed to create thread in Elastic: %v", err)
			}
		}
	}

	time.Sleep(500 * time.Millisecond)

	agentsCtrl := agents.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r.GET("/api/v1/organisations/:org_id/agents", middleware.Authorize(db.Postgresql), agentsCtrl.FetchOrganisationAgents)

	expectedAgentOne := fmt.Sprintf("Agent One %s", currUUID)
	expectedAgentTwo := fmt.Sprintf("Agent Two %s", currUUID)

	t.Run("Successfully Get Agents with Previews and Correct Sort", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/organisations/%s/agents", orgID)
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)

		responseData := data["data"].([]interface{})
		if len(responseData) < 2 {
			t.Fatalf("Expected at least 2 agents in response, got %d", len(responseData))
		}

		firstAgent := responseData[0].(map[string]interface{})
		if firstAgent["name"] != expectedAgentOne {
			t.Errorf("Expected %s as first agent (newest message), got %s", expectedAgentOne, firstAgent["name"])
		}

		secondAgent := responseData[1].(map[string]interface{})
		if secondAgent["name"] != expectedAgentTwo {
			t.Errorf("Expected %s as second agent, got %s", expectedAgentTwo, secondAgent["name"])
		}

		if firstAgent["preview_message"] != "Hello from Agent One" {
			t.Errorf("Expected preview_message 'Hello from Agent One', got %s", firstAgent["preview_message"])
		}

		previewThread := firstAgent["preview_thread"].([]interface{})
		if len(previewThread) == 0 {
			t.Error("Expected preview_thread to be populated")
		} else {
			firstThread := previewThread[0].(map[string]interface{})
			if firstThread["message"] != "Hello from Agent One" {
				t.Errorf("Expected message in preview_thread 'Hello from Agent One', got %s", firstThread["message"])
			}
		}

		participants := firstAgent["participants"].([]interface{})
		if len(participants) != 2 {
			t.Errorf("Expected 2 participants, got %d", len(participants))
		}

		agentParticipantFound := false
		for _, p := range participants {
			part := p.(map[string]interface{})
			if part["user_type"] == "bot" {
				agentParticipantFound = true
				expectedEmail := fmt.Sprintf("%s@telex.im", slug.Make(firstAgent["name"].(string)))
				if part["email"] != expectedEmail {
					t.Errorf("Expected bot email %s, got %s", expectedEmail, part["email"])
				}
				if part["username"] != firstAgent["name"] {
					t.Errorf("Expected bot username %s, got %s", firstAgent["name"], part["username"])
				}
			}
		}

		if !agentParticipantFound {
			t.Error("Bot participant not found in participants list")
		}
	})
}
