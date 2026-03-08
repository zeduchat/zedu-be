package test_message

import (
	"encoding/json"
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
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSystemMessagesForDMOperations(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("sysuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "System",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("sysuser1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("sysuser2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "System",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("sysuser2_%v", currUUID),
	}

	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("sysuser3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "System",
		LastName:    "User3",
		Password:    "password",
		UserName:    fmt.Sprintf("sysuser3_%v", currUUID),
	}

	user4SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("sysuser4_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "System",
		LastName:    "User4",
		Password:    "password",
		UserName:    fmt.Sprintf("sysuser4_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
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
	tst.SignupUser(t, r, authController, user1SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user2SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user3SignUpData, false)
	tst.SignupUser(t, gin.Default(), authController, user4SignUpData, false)
	token1 := tst.GetLoginToken(t, r, authController, loginData1)

	var user1, user2, user3, user4 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user3SignUpData.Email).First(&user3).Error; err != nil {
		t.Fatalf("Failed to get user3: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user4SignUpData.Email).First(&user4).Error; err != nil {
		t.Fatalf("Failed to get user4: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	t.Run("DM Channel Creation - System Message", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/dms", middleware.Authorize(db.Postgresql), controller.CreateDmChannel)

		reqBody := fmt.Sprintf(`{"participant_id": "%s", "chat_type": "user"}`, user2.ID)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/dms", org.ID), strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response missing data field")
		}

		channelID, ok := data["channel_id"].(string)
		if !ok {
			t.Fatal("Channel ID not found in response")
		}

		time.Sleep(5 * time.Second)

		systemMessage := tst.FetchSystemMessage(t, db, logger, channelID, token1)
		if systemMessage == nil {
			t.Error("No system message found for DM channel creation")
			return
		}

		content, ok := systemMessage["message"].(string)
		if !ok {
			t.Error("System message content is missing")
			return
		}

		expectedContent := fmt.Sprintf("<p>started a conversation with <span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> </p><p></p>", user2.ID, user2SignUpData.UserName, user2SignUpData.UserName)
		if content != expectedContent {
			t.Errorf("System message content mismatch. Expected: '%s', Got: '%s'", expectedContent, content)
		} else {
			t.Logf("System message created correctly: '%s'", content)
		}
	})

	t.Run("Group DM Creation - Concatenated System Message", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/group-dms", middleware.Authorize(db.Postgresql), controller.CreateGroupDMChannel)

		reqBody := fmt.Sprintf(`{"participants": ["%s", "%s", "%s"], "chat_type": "user"}`, user2.ID, user3.ID, user4.ID)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/group-dms", org.ID), strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response missing data field")
		}

		channelID, ok := data["channel_id"].(string)
		if !ok {
			t.Fatal("Channel ID not found in response")
		}

		time.Sleep(5 * time.Second)

		systemMessage := tst.FetchSystemMessage(t, db, logger, channelID, token1)
		if systemMessage == nil {
			t.Error("No system message found for group DM creation")
			return
		}

		content, ok := systemMessage["message"].(string)
		if !ok {
			t.Error("System message content is missing")
			return
		}

		if !strings.Contains(content, "joined the group") {
			t.Errorf("System message doesn't contain 'joined the group'. Got: '%s'", content)
		}

		usernames := []string{
			fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", user1.ID, user1SignUpData.UserName, user1SignUpData.UserName),
			fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", user2.ID, user2SignUpData.UserName, user2SignUpData.UserName),
			fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", user3.ID, user3SignUpData.UserName, user3SignUpData.UserName),
			fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", user4.ID, user4SignUpData.UserName, user4SignUpData.UserName),
		}

		for _, usernameSpan := range usernames {
			if !strings.Contains(content, usernameSpan) {
				t.Errorf("System message doesn't contain username span '%s'", usernameSpan)
			}
		}

	})

	t.Run("Join Group DM - System Message", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID,
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel: %v", err)
		}

		participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    user1.ID,
			OrgId:     org.ID,
		}
		if err := db.Postgresql.Create(&participant).Error; err != nil {
			t.Fatalf("Failed to create channel participant: %v", err)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/group-dms/:channel_id/join", middleware.Authorize(db.Postgresql), controller.JoinGroupDMChannel)

		reqBody := `{}`
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/join", org.ID, groupDMChannelID), strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		loginData2 := models.LoginRequestModel{
			Email:    user2SignUpData.Email,
			Password: user2SignUpData.Password,
		}
		token2 := tst.GetLoginToken(t, gin.Default(), authController, loginData2)

		req.Header.Set("Authorization", "Bearer "+token2)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		time.Sleep(3 * time.Second)

		systemMessage := tst.FetchSystemMessage(t, db, logger, groupDMChannelID, token2)
		if systemMessage == nil {
			t.Error("No system message found for joining group DM")
			return
		}

		content, ok := systemMessage["message"].(string)
		if !ok {
			t.Error("System message content is missing")
			return
		}

		expectedContent := fmt.Sprintf("<p><span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> joined the group</p><p></p>", user2.ID, user2SignUpData.UserName, user2SignUpData.UserName)
		if content != expectedContent {
			t.Errorf("System message content mismatch. Expected: '%s', Got: '%s'", expectedContent, content)
		} else {
			t.Logf("System message created correctly for join: '%s'", content)
		}
	})

	t.Run("Leave Group DM - System Message", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		users := []models.User{user1, user2}
		for _, user := range users {
			groupDMChannel := models.DmChannels{
				ID:              utility.GenerateUUID(),
				UserId:          user.ID,
				ChannelId:       groupDMChannelID,
				OrgId:           org.ID,
				ParticipantHash: participantHash,
				ChatType:        "user",
				ChannelType:     "group_dm",
			}
			if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
				t.Fatalf("Failed to create group DM channel: %v", err)
			}

			participant := models.ChannelParticipant{
				ID:        utility.GenerateUUID(),
				ChannelId: groupDMChannelID,
				UserId:    user.ID,
				OrgId:     org.ID,
			}
			if err := db.Postgresql.Create(&participant).Error; err != nil {
				t.Fatalf("Failed to create channel participant: %v", err)
			}
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.DELETE("/api/v1/organisations/:org_id/group-dms/:channel_id/leave", middleware.Authorize(db.Postgresql), controller.LeaveGroupDMChannel)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/leave", org.ID, groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		time.Sleep(3 * time.Second)

		systemMessage := tst.FetchSystemMessage(t, db, logger, groupDMChannelID, token1)
		if systemMessage == nil {
			t.Error("No system message found for leaving group DM")
			return
		}

		content, ok := systemMessage["message"].(string)
		if !ok {
			t.Error("System message content is missing")
			return
		}

		expectedContent := fmt.Sprintf("<p><span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span> left the group</p><p></p>", user1.ID, user1SignUpData.UserName, user1SignUpData.UserName)
		if content != expectedContent {
			t.Errorf("System message content mismatch. Expected: '%s', Got: '%s'", expectedContent, content)
		} else {
			t.Logf("System message created correctly for leave: '%s'", content)
		}
	})

	t.Run("Add Participants - Concatenated System Message", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user1.ID,
			ChannelId:       groupDMChannelID,
			OrgId:           org.ID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDMChannel).Error; err != nil {
			t.Fatalf("Failed to create group DM channel: %v", err)
		}

		participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    user1.ID,
			OrgId:     org.ID,
		}
		if err := db.Postgresql.Create(&participant).Error; err != nil {
			t.Fatalf("Failed to create channel participant: %v", err)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/:org_id/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		reqBody := fmt.Sprintf(`{"user_ids": ["%s", "%s"]}`, user2.ID, user3.ID)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/participants", org.ID, groupDMChannelID), strings.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
			return
		}

		time.Sleep(3 * time.Second)

		systemMessage := tst.FetchSystemMessage(t, db, logger, groupDMChannelID, token1)
		if systemMessage == nil {
			t.Error("No system message found for adding participants")
			return
		}

		content, ok := systemMessage["message"].(string)
		if !ok {
			t.Error("System message content is missing")
			return
		}

		if !strings.Contains(content, "joined the group") {
			t.Errorf("System message doesn't contain 'joined the group'. Got: '%s'", content)
		}

		expectedUsernames := []string{
			fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", user2.ID, user2SignUpData.UserName, user2SignUpData.UserName),
			fmt.Sprintf("<span class=\"mention\" data-type=\"mention\" data-id=\"%s\" data-label=\"%s\" data-mention-suggestion-char=\"@\">@%s</span>", user3.ID, user3SignUpData.UserName, user3SignUpData.UserName),
		}

		for _, usernameSpan := range expectedUsernames {
			if !strings.Contains(content, usernameSpan) {
				t.Errorf("System message doesn't contain username span '%s'", usernameSpan)
			}
		}

		if strings.Contains(content, " and ") {
			t.Logf("System message correctly formatted with concatenated usernames: '%s'", content)
		} else {
			t.Errorf("System message not properly concatenated. Got: '%s'", content)
		}
	})
}
