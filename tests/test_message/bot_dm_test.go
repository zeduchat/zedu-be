package test_message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/openrouter"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBotDirectMessageWithToolCall(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Initialize OpenRouter client for test
	openRouterCfg := config.OpenRouter{
		ApiKey:  "test_key",
		BaseUrl: "http://test-url",
	}
	openrouter.NewOpenRouterClient(logger, openRouterCfg, nil)

	defer tst.Cleanup(db)

	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("botuser_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "BotUser",
		LastName:    "Tester",
		Password:    "password",
		UserName:    fmt.Sprintf("botuser_%v", currUUID),
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

	r := gin.Default()
	tst.SignupUser(t, r, authController, userSignUpData, false)

	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}
	token := tst.GetLoginToken(t, r, authController, loginData)

	var user models.User
	if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	agentID := utility.GenerateUUID()
	integration := models.Integrations{
		ID:             agentID,
		Name:           fmt.Sprintf("Test Agent %s", agentID),
		AppDescription: "A test agent",
		IsActive:       true,
		IsSystem:       true,
		OwnerID:        user.ID,
		PreSharedKey:   utility.GenerateUUID(),
	}
	if err := db.Postgresql.Create(&integration).Error; err != nil {
		t.Fatalf("Failed to create integration: %v", err)
	}

	orgIntegration := models.OrganisationIntegrations{
		ID:            utility.GenerateUUID(),
		OrgID:         org.ID,
		IntegrationID: agentID,
		OwnerID:       user.ID,
		IsActive:      true,
		IsSystem:      true,
	}
	if err := db.Postgresql.Create(&orgIntegration).Error; err != nil {
		t.Fatalf("Failed to create org integration: %v", err)
	}

	dmChannelID := utility.GenerateUUID()
	botID := agentID

	dmChannel := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        user.ID,
		ChannelId:     dmChannelID,
		OrgId:         org.ID,
		ParticipantId: &botID,
		ChatType:      "bot",
		ChannelType:   "dm",
	}

	channel := models.Channels{
		ID:             dmChannelID,
		Name:           "Bot DM Channel",
		Description:    "Direct message with bot",
		OrganisationID: org.ID,
		OwnerId:        user.ID,
		CreatedAt:      time.Now(),
	}
	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
		t.Fatalf("Failed to create Bot DM channel: %v", err)
	}

	userChannel := models.UserChannels{
		ChannelsID: dmChannelID,
		UserID:     user.ID,
		Username:   userSignUpData.UserName,
	}
	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Fatalf("Failed to add user to channel: %v", err)
	}

	extReq := request.ExternalRequest{Logger: logger, Test: true}
	controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

	threadController := thread.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

	r.POST("/api/v1/dms/channels/:channel_id/threads", controller.AddAThreadDm)

	t.Run("Send Message Triggering Tool Call", func(t *testing.T) {
		t.Skip("Bot DM dispatch is currently disabled")
		reqBody := models.CreateThreadMsgReq{
			Content: "Please capitalize this text: hello world",
			Type:    "message",
		}

		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/dms/channels/%s/threads", dmChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		router := gin.Default()
		dmGroup := router.Group("/api/v1/dms", middleware.Authorize(db.Postgresql))
		{
			dmGroup.POST("/channels/:channel_id/threads", controller.AddAThreadDm)
		}

		threadGroup := router.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
		{
			threadGroup.GET("/channels/:channel_id/threads", threadController.GetAllChannelThreads)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201 Created, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		time.Sleep(2 * time.Second)

		threadReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/threads/channels/%s/threads", dmChannelID), nil)
		threadReq.Header.Set("Authorization", "Bearer "+token)
		threadReq.Header.Set("Content-Type", "application/json")

		threadRR := httptest.NewRecorder()
		router.ServeHTTP(threadRR, threadReq)

		if threadRR.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK for fetching threads, got %d. Response: %s", threadRR.Code, threadRR.Body.String())
		}

		var threadResponse map[string]interface{}
		if err := json.Unmarshal(threadRR.Body.Bytes(), &threadResponse); err != nil {
			t.Fatalf("Failed to parse thread response: %v", err)
		}

		data, ok := threadResponse["data"].([]interface{})
		if !ok {
			t.Fatalf("Expected data to be an array")
		}

		foundBotMessage := false
		for _, msg := range data {
			msgMap, ok := msg.(map[string]interface{})
			if !ok {
				continue
			}
			content, ok := msgMap["message"].(string)
			if ok && content == "HELLO WORLD" {
				foundBotMessage = true
				break
			}
		}

		if !foundBotMessage {
			t.Errorf("Expected to find bot message 'HELLO WORLD' in channel threads")
		}

	})
}
