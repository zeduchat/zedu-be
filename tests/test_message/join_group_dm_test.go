package test_message

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
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestJoinGroupDMChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserOne",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserTwo",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_user2_%v", currUUID),
	}

	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserThree",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_user3_%v", currUUID),
	}

	user4SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_user4_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserFour",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_user4_%v", currUUID),
	}

	loginData4 := models.LoginRequestModel{
		Email:    user4SignUpData.Email,
		Password: user4SignUpData.Password,
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
	token4 := tst.GetLoginToken(t, gin.Default(), authController, loginData4)

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

	t.Run("Successfully Join Group DM Channel", func(t *testing.T) {
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

		for _, user := range []models.User{user1, user2, user3} {
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
		r.POST("/api/v1/organisations/group-dms/:channel_id/join", middleware.Authorize(db.Postgresql), controller.JoinGroupDMChannel)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/join", groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token4)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
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
		if !ok || channelID != groupDMChannelID {
			t.Errorf("Expected channel_id %s, got %v", groupDMChannelID, data["channel_id"])
		}

		channelType, ok := data["channel_type"].(string)
		if !ok || channelType != "group_dm" {
			t.Errorf("Expected channel_type 'group_dm', got %v", data["channel_type"])
		}

		participants, ok := data["participants"].([]interface{})
		if !ok {
			t.Fatal("participants field is missing or not an array")
		}

		if len(participants) != 4 {
			t.Errorf("Expected 4 participants after join, got %d", len(participants))
		}

		foundUser4 := false
		for _, p := range participants {
			participant, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if participant["user_id"] == user4.ID {
				foundUser4 = true
				t.Logf("✅ User4 successfully added to participants list")
				break
			}
		}

		if !foundUser4 {
			t.Error("User4 not found in participants list after joining")
		}

		var participantCount int64
		db.Postgresql.Model(&models.ChannelParticipant{}).Where("channel_id = ?", groupDMChannelID).Count(&participantCount)
		if participantCount != 4 {
			t.Errorf("Expected 4 participants in database, got %d", participantCount)
		}

		t.Logf("✅ User successfully joined group DM channel")
	})

	t.Run("Fail to Join - User Already a Participant", func(t *testing.T) {
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

		for _, user := range []models.User{user1, user2, user3, user4} {
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
		r.POST("/api/v1/organisations/group-dms/:channel_id/join", middleware.Authorize(db.Postgresql), controller.JoinGroupDMChannel)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/join", groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token4)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		message, ok := response["message"].(string)
		if !ok {
			t.Fatal("Response missing message field")
		}

		if message != "user is already a participant in the group DM channel" {
			t.Errorf("Expected conflict message, got: %s", message)
		}

		t.Logf("✅ Correctly prevented duplicate join")
	})

	t.Run("Fail to Join - Channel Does Not Exist", func(t *testing.T) {
		nonExistentChannelID := utility.GenerateUUID()

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/group-dms/:channel_id/join", middleware.Authorize(db.Postgresql), controller.JoinGroupDMChannel)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/join", nonExistentChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token4)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		message, ok := response["message"].(string)
		if !ok {
			t.Fatal("Response missing message field")
		}

		if message != "group DM channel does not exist" {
			t.Errorf("Expected not found message, got: %s", message)
		}

		t.Logf("✅ Correctly handled non-existent channel")
	})

	t.Run("Fail to Join - Channel at Maximum Capacity", func(t *testing.T) {
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

		user5SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupdm_user5_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "GroupDM",
			LastName:    "UserFive",
			Password:    "password",
			UserName:    fmt.Sprintf("groupdm_user5_%v", currUUID),
		}

		user6SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupdm_user6_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "GroupDM",
			LastName:    "UserSix",
			Password:    "password",
			UserName:    fmt.Sprintf("groupdm_user6_%v", currUUID),
		}

		tst.SignupUser(t, gin.Default(), authController, user5SignUpData, false)
		tst.SignupUser(t, gin.Default(), authController, user6SignUpData, false)

		var user5, user6 models.User
		if err := db.Postgresql.Where("email = ?", user5SignUpData.Email).First(&user5).Error; err != nil {
			t.Fatalf("Failed to get user5: %v", err)
		}
		if err := db.Postgresql.Where("email = ?", user6SignUpData.Email).First(&user6).Error; err != nil {
			t.Fatalf("Failed to get user6: %v", err)
		}

		for _, user := range []models.User{user1, user2, user3, user4, user5, user6} {
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

		user7SignUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("groupdm_user7_%v@qa.team", currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "GroupDM",
			LastName:    "UserSeven",
			Password:    "password",
			UserName:    fmt.Sprintf("groupdm_user7_%v", currUUID),
		}

		loginData7 := models.LoginRequestModel{
			Email:    user7SignUpData.Email,
			Password: user7SignUpData.Password,
		}

		tst.SignupUser(t, gin.Default(), authController, user7SignUpData, false)
		token7 := tst.GetLoginToken(t, gin.Default(), authController, loginData7)

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/group-dms/:channel_id/join", middleware.Authorize(db.Postgresql), controller.JoinGroupDMChannel)

		reqBody := bytes.NewBuffer([]byte(`{}`))
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/join", groupDMChannelID), reqBody)
		req.Header.Set("Authorization", "Bearer "+token7)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		message, ok := response["message"].(string)
		if !ok {
			t.Fatal("Response missing message field")
		}

		if message != "group DM channel has reached maximum capacity of 6 participants" {
			t.Errorf("Expected capacity message, got: %s", message)
		}

		t.Logf("✅ Correctly prevented join when at maximum capacity")
	})
}
