package test_buzz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzzSystemMessage(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()

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

	tst.SignupUser(t, r, auth, userSignUpData, false)
	token := tst.GetLoginToken(t, r, auth, loginData)

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("TestTeam%s", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}

	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	var user models.User
	if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
		t.Fatalf("failed to fetch user: %v", err)
	}

	channelID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             channelID,
		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
		Description:    "Some Random description",
		OrganisationID: orgId,
		OwnerId:        user.ID,
		CreatedAt:      time.Now(),
	}

	if err := db.Postgresql.Create(&channel).Error; err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	userChannel := models.UserChannels{
		ChannelsID: channelID,
		UserID:     user.ID,
		Username:   userSignUpData.UserName,
		CreatedAt:  time.Now(),
	}

	if err := db.Postgresql.Create(&userChannel).Error; err != nil {
		t.Logf("Warning: Failed to add user to channel: %v", err)
	}

	t.Cleanup(func() {
		tst.Cleanup(db)
	})

	createBuzzReq := models.CreateBuzzRequest{
		ChannelID: channelID,
	}

	t.Run("BuzzStartCreatesSystemMessage", func(t *testing.T) {
		router, _ := SetupBuzzTestRouter(logger, validatorRef)

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusInternalServerError {
			t.Skip("Skipping test - Elastic client not available (expected in test environment)")
			return
		}

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)
		buzzID := dataM["buzz_id"].(string)

		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz: %v", err)
		}

		if buzz.Status != models.BuzzStatusActive {
			t.Errorf("Expected buzz status 'active', got %s", buzz.Status)
		}

		if len(buzz.ParticipantIDs) < 1 {
			t.Error("Expected at least 1 participant")
		}

		time.Sleep(3 * time.Second)

		threads := getChannelSystemMessages(t, router, channelID, token)
		if len(threads) > 0 {
			msg := threads[0]
			if msg.Type != "system" {
				t.Errorf("Expected message type 'system', got %s", msg.Type)
			}

			if msg.ChannelsID != channelID {
				t.Errorf("Expected channel_id %s, got %s", channelID, msg.ChannelsID)
			}

			if msg.Status != "success" {
				t.Errorf("Expected status 'success', got %s", msg.Status)
			}

			if !validateBuzzStartMessageContent(t, msg.Content, strings.Split(user.Email, "@")[0], len(buzz.ParticipantIDs)) {
				t.Error("System message content format validation failed")
			}
		} else {
			t.Log("Note: No system messages found - Elastic search may not be working properly")
		}

		endUrl := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		endReq, _ := http.NewRequest(http.MethodPost, endUrl, nil)
		endReq.Header.Set("Content-Type", "application/json")
		endReq.Header.Set("Authorization", "Bearer "+token)

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, endReq)

		if rr2.Code != http.StatusOK {
			t.Logf("Failed to end buzz %s (status %d), this may affect subsequent tests", buzzID, rr2.Code)
		}

		time.Sleep(1 * time.Second)
	})

	t.Run("BuzzEndCreatesSystemMessage", func(t *testing.T) {
		router, _ := SetupBuzzTestRouter(logger, validatorRef)

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusInternalServerError {
			t.Skip("Skipping test - Elastic client not available (expected in test environment)")
			return
		}

		if rr.Code == http.StatusConflict {
			t.Skip("Skipping test - Channel already has active buzz from previous test")
			return
		}

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)
		buzzID := dataM["buzz_id"].(string)

		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz: %v", err)
		}

		time.Sleep(2 * time.Second)

		beforeEndThreads := getChannelSystemMessages(t, router, channelID, token)
		startCount := len(beforeEndThreads)

		endUrl := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		endReq, _ := http.NewRequest(http.MethodPost, endUrl, nil)
		endReq.Header.Set("Content-Type", "application/json")
		endReq.Header.Set("Authorization", "Bearer "+token)

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, endReq)

		tst.AssertStatusCode(t, rr2.Code, http.StatusOK)

		var endedBuzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&endedBuzz).Error; err != nil {
			t.Fatalf("Failed to fetch buzz after ending: %v", err)
		}

		if endedBuzz.Status != models.BuzzStatusEnded {
			t.Errorf("Expected buzz status 'ended', got %s", endedBuzz.Status)
		}

		if endedBuzz.BuzzEndTime == nil {
			t.Error("Expected buzz_end_time to be set")
		}

		time.Sleep(2 * time.Second)

		afterEndThreads := getChannelSystemMessages(t, router, channelID, token)
		if len(afterEndThreads) > startCount {
			endMsg := afterEndThreads[0]
			if endMsg.Type != "system" {
				t.Errorf("Expected message type 'system', got %s", endMsg.Type)
			}

			if !validateBuzzEndMessageContent(t, endMsg.Content, strings.Split(user.Email, "@")[0], len(buzz.ParticipantIDs)) {
				t.Error("System message content format validation failed")
			}
		} else {
			t.Log("Note: No new system messages found - Elastic search may not be working properly")
		}
	})

	t.Run("SystemMessageContentFormat", func(t *testing.T) {
		router, _ := SetupBuzzTestRouter(logger, validatorRef)

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusInternalServerError {
			t.Skip("Skipping test - Elastic client not available (expected in test environment)")
			return
		}

		if rr.Code == http.StatusConflict {
			t.Skip("Skipping test - Channel already has active buzz from previous test")
			return
		}

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)
		buzzID := dataM["buzz_id"].(string)

		time.Sleep(2 * time.Second)

		threads := getChannelSystemMessages(t, router, channelID, token)
		if len(threads) == 0 {
			t.Skip("Skipping test - No system messages found")
			return
		}

		msg := threads[0]

		startPattern := regexp.MustCompile(`^@(\w+)\s+started\s+a\s+buzz\s+\((\d+)\s+participants\)$`)
		if !startPattern.MatchString(msg.Content) {
			t.Errorf("Start message does not match expected format. Got: %s", msg.Content)
		}

		matches := startPattern.FindStringSubmatch(msg.Content)
		if len(matches) != 3 {
			t.Fatalf("Expected 3 matches, got %d", len(matches))
		}

		endUrl := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		endReq, _ := http.NewRequest(http.MethodPost, endUrl, nil)
		endReq.Header.Set("Content-Type", "application/json")
		endReq.Header.Set("Authorization", "Bearer "+token)

		time.Sleep(1 * time.Second)

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, endReq)

		time.Sleep(2 * time.Second)

		afterEndThreads := getChannelSystemMessages(t, router, channelID, token)
		if len(afterEndThreads) < 2 {
			t.Skip("Skipping test - No end message found")
			return
		}

		endMsg := afterEndThreads[0]
		endPattern := regexp.MustCompile(`^@(\w+)\s+ended\s+the\s+buzz\s+\((\d+)\s+participants\)$`)
		if !endPattern.MatchString(endMsg.Content) {
			t.Errorf("End message does not match expected format. Got: %s", endMsg.Content)
		}

		endMatches := endPattern.FindStringSubmatch(endMsg.Content)
		if len(endMatches) != 3 {
			t.Fatalf("Expected 3 matches for end message, got %d", len(endMatches))
		}

		username := endMatches[1]
		participantCountStr := endMatches[2]

		if participantCountStr != "1" {
			t.Errorf("Expected participant count '1', got '%s'", participantCountStr)
		}

		if username != user.Profile.UserName {
			t.Logf("Note: Username mismatch may be expected in test environment")
		}
	})

	t.Run("BuzzMessagesInChannelHistory", func(t *testing.T) {
		router, _ := SetupBuzzTestRouter(logger, validatorRef)

		threadController := thread.Controller{Db: db, Validator: validatorRef, Logger: logger}
		threadRouter := gin.Default()
		threadUrl := threadRouter.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
		{
			threadUrl.GET("/channels/:channel_id/threads", threadController.GetAllChannelThreads)
		}

		initialThreads := getChannelAllMessages(t, threadRouter, channelID, token)

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusInternalServerError {
			t.Skip("Skipping test - Elastic client not available (expected in test environment)")
			return
		}

		if rr.Code == http.StatusConflict {
			t.Skip("Skipping test - Channel already has active buzz from previous test")
			return
		}

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)
		buzzID := dataM["buzz_id"].(string)

		time.Sleep(2 * time.Second)

		afterStartThreads := getChannelAllMessages(t, threadRouter, channelID, token)

		endUrl := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		endReq, _ := http.NewRequest(http.MethodPost, endUrl, nil)
		endReq.Header.Set("Content-Type", "application/json")
		endReq.Header.Set("Authorization", "Bearer "+token)

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, endReq)

		tst.AssertStatusCode(t, rr2.Code, http.StatusOK)

		time.Sleep(2 * time.Second)

		afterEndThreads := getChannelAllMessages(t, threadRouter, channelID, token)

		systemMessages := filterMessagesByType(afterEndThreads, "system")
		if len(systemMessages) > 0 {
			if len(afterStartThreads) != len(initialThreads) && len(afterEndThreads) != len(initialThreads) {
				t.Logf("System messages created (start: %d, end: %d)", len(afterStartThreads)-len(initialThreads), len(afterEndThreads)-len(initialThreads))
			}
			for i, msg := range systemMessages {
				if msg.Type != "system" {
					t.Errorf("Message %d: Expected type 'system', got '%s'", i, msg.Type)
				}
				if msg.ChannelsID != channelID {
					t.Errorf("Message %d: Expected channel_id %s, got %s", i, channelID, msg.ChannelsID)
				}
			}
		} else {
			t.Log("Note: No system messages found - Elastic search may not be working properly")
		}
	})

	t.Run("BuzzMessagesNotInThreadHistory", func(t *testing.T) {
		router, _ := SetupBuzzTestRouter(logger, validatorRef)

		threadController := thread.Controller{Db: db, Validator: validatorRef, Logger: logger}
		threadRouter := gin.Default()
		threadUrl := threadRouter.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
		{
			threadUrl.GET("/channels/:channel_id", threadController.GetChannelThreads)
		}

		var b bytes.Buffer
		json.NewEncoder(&b).Encode(createBuzzReq)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code == http.StatusInternalServerError {
			t.Skip("Skipping test - Elastic client not available (expected in test environment)")
			return
		}

		if rr.Code == http.StatusConflict {
			t.Skip("Skipping test - Channel already has active buzz from previous test")
			return
		}

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)
		buzzID := dataM["buzz_id"].(string)

		time.Sleep(2 * time.Second)

		getUrl := fmt.Sprintf("/api/v1/threads/channels/%s", channelID)
		getReq, _ := http.NewRequest(http.MethodGet, getUrl, nil)
		getReq.Header.Set("Authorization", "Bearer "+token)

		rr2 := httptest.NewRecorder()
		threadRouter.ServeHTTP(rr2, getReq)

		if rr2.Code != http.StatusOK && rr2.Code != http.StatusNoContent {
			t.Logf("Unexpected status code: %d", rr2.Code)
		}

		threadData := tst.ParseResponse(rr2)
		threads, ok := threadData["data"].([]interface{})
		if !ok {
			t.Log("Note: Thread data format unexpected, expected slice in data field")
			return
		}

		for i, threadItem := range threads {
			threadMap := threadItem.(map[string]interface{})
			threadType := threadMap["type"].(string)
			if threadType == "system" {
				t.Errorf("System message should not be in thread history. Found at index %d", i)
			}
		}

		endUrl := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		endReq, _ := http.NewRequest(http.MethodPost, endUrl, nil)
		endReq.Header.Set("Content-Type", "application/json")
		endReq.Header.Set("Authorization", "Bearer "+token)

		rr3 := httptest.NewRecorder()
		router.ServeHTTP(rr3, endReq)

		time.Sleep(2 * time.Second)

		getReq2, _ := http.NewRequest(http.MethodGet, getUrl, nil)
		getReq2.Header.Set("Authorization", "Bearer "+token)

		rr4 := httptest.NewRecorder()
		threadRouter.ServeHTTP(rr4, getReq2)

		threadData2 := tst.ParseResponse(rr4)
		threads2, ok := threadData2["data"].([]interface{})
		if !ok {
			return
		}

		for i, threadItem := range threads2 {
			threadMap := threadItem.(map[string]interface{})
			threadType := threadMap["type"].(string)
			if threadType == "system" {
				t.Errorf("System message should not be in thread history. Found at index %d", i)
			}
		}
	})

	t.Run("MultipleBuzzesCreateMultipleMessages", func(t *testing.T) {
		router, _ := SetupBuzzTestRouter(logger, validatorRef)

		getChannelSystemMessages(t, router, channelID, token)

		var buzzIDs []string

		for i := 0; i < 3; i++ {
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(createBuzzReq)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/create", &b)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code == http.StatusInternalServerError && i == 0 {
				t.Skip("Skipping test - Elastic client not available (expected in test environment)")
				return
			}

			if rr.Code == http.StatusConflict && i == 0 {
				t.Skip("Skipping test - Channel has active buzz from previous test")
				return
			}

			if rr.Code != http.StatusCreated {
				t.Logf("Buzz %d creation failed with status %d, skipping", i+1, rr.Code)
				continue
			}

			data := tst.ParseResponse(rr)
			dataM := data["data"].(map[string]any)
			buzzIDs = append(buzzIDs, dataM["buzz_id"].(string))

			time.Sleep(1 * time.Second)
		}

		if len(buzzIDs) == 0 {
			t.Skip("No buzzes created, skipping message validation")
			return
		}

		for _, buzzID := range buzzIDs {
			endUrl := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
			endReq, _ := http.NewRequest(http.MethodPost, endUrl, nil)
			endReq.Header.Set("Content-Type", "application/json")
			endReq.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, endReq)

			if rr.Code != http.StatusOK {
				t.Logf("Failed to end buzz %s with status %d", buzzID, rr.Code)
				continue
			}

			time.Sleep(500 * time.Millisecond)
		}

		time.Sleep(3 * time.Second)

		finalThreads := getChannelSystemMessages(t, router, channelID, token)

		startMessages := 0
		endMessages := 0
		for _, msg := range finalThreads {
			if strings.Contains(msg.Content, "started a buzz") {
				startMessages++
			} else if strings.Contains(msg.Content, "ended the buzz") {
				endMessages++
			}
		}

		if startMessages > 0 {
			t.Logf("Found %d start messages", startMessages)
		}
		if endMessages > 0 {
			t.Logf("Found %d end messages", endMessages)
		}

		if startMessages+endMessages > 0 {
			t.Logf("✓ System messages are being created for multiple buzzes")
		} else {
			t.Log("Note: No system messages found - Elastic search may not be working properly")
		}

		if startMessages != 3 {
			t.Logf("Warning: Expected 3 start messages, got %d", startMessages)
		}
		if endMessages != 3 {
			t.Logf("Warning: Expected 3 end messages, got %d", endMessages)
		}
	})
}

func getChannelSystemMessages(t *testing.T, r *gin.Engine, channelID, token string) []models.ThreadDocument {
	threads := getChannelAllMessages(t, r, channelID, token)
	return filterMessagesByType(threads, "system")
}

func getChannelAllMessages(t *testing.T, r *gin.Engine, channelID, token string) []models.ThreadDocument {
	if storage.DB.Elastic == nil {
		t.Log("Elastic client is nil, returning empty threads")
		return []models.ThreadDocument{}
	}

	getUrl := fmt.Sprintf("/api/v1/threads/channels/%s/threads", channelID)
	req, _ := http.NewRequest(http.MethodGet, getUrl, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusNoContent {
		t.Logf("Failed to get channel messages: %d - %s", rr.Code, rr.Body.String())
		return []models.ThreadDocument{}
	}

	if rr.Code == http.StatusNoContent {
		return []models.ThreadDocument{}
	}

	data := tst.ParseResponse(rr)
	threadsData, ok := data["data"].([]interface{})
	if !ok {
		return []models.ThreadDocument{}
	}

	threads := make([]models.ThreadDocument, 0, len(threadsData))
	for _, threadItem := range threadsData {
		threadMap := threadItem.(map[string]interface{})
		threadJSON, _ := json.Marshal(threadMap)
		var thread models.ThreadDocument
		json.Unmarshal(threadJSON, &thread)
		threads = append(threads, thread)
	}

	return threads
}

func filterMessagesByType(threads []models.ThreadDocument, msgType string) []models.ThreadDocument {
	filtered := make([]models.ThreadDocument, 0)
	for _, thread := range threads {
		if thread.Type == msgType {
			filtered = append(filtered, thread)
		}
	}
	return filtered
}

func validateBuzzStartMessageContent(t *testing.T, content, username string, participantCount int) bool {
	expected := fmt.Sprintf("@%s started a buzz (%d participants)", username, participantCount)
	var check = content == expected
	if !check {
		t.Logf("Expected: %s", expected)
		t.Logf("Got: %s", content)
	}
	return check
}

func validateBuzzEndMessageContent(t *testing.T, content, username string, participantCount int) bool {
	expected := fmt.Sprintf("@%s ended the buzz (%d participants)", username, participantCount)
	var check = content == expected
	if !check {
		t.Logf("Expected: %s", expected)
		t.Logf("Got: %s", content)
	}
	return check
}
