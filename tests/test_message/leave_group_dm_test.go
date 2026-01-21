package test_message

import (
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

func TestLeaveGroupDMChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("leave_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Leave",
		LastName:    "UserOne",
		Password:    "password",
		UserName:    fmt.Sprintf("leave_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("leave_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Leave",
		LastName:    "UserTwo",
		Password:    "password",
		UserName:    fmt.Sprintf("leave_user2_%v", currUUID),
	}

	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("leave_user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Leave",
		LastName:    "UserThree",
		Password:    "password",
		UserName:    fmt.Sprintf("leave_user3_%v", currUUID),
	}

	user4SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("leave_user4_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Leave",
		LastName:    "UserFour",
		Password:    "password",
		UserName:    fmt.Sprintf("leave_user4_%v", currUUID),
	}

	loginData3 := models.LoginRequestModel{
		Email:    user3SignUpData.Email,
		Password: user3SignUpData.Password,
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
	token3 := tst.GetLoginToken(t, gin.Default(), authController, loginData3)
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

	t.Run("Successfully Leave Group DM Channel", func(t *testing.T) {
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

		var initialCount int64
		db.Postgresql.Model(&models.ChannelParticipant{}).Where("channel_id = ?", groupDMChannelID).Count(&initialCount)
		if initialCount != 3 {
			t.Fatalf("Expected 3 initial participants, got %d", initialCount)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.DELETE("/api/v1/organisations/:org_id/group-dms/:channel_id/leave", middleware.Authorize(db.Postgresql), controller.LeaveGroupDMChannel)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/leave", org.ID, groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		message, ok := response["message"].(string)
		if !ok {
			t.Fatal("Response missing message field")
		}

		if message != "User left Group DM channel successfully" {
			t.Errorf("Expected success message, got: %s", message)
		}

		var participantCount int64
		db.Postgresql.Model(&models.ChannelParticipant{}).Where("channel_id = ?", groupDMChannelID).Count(&participantCount)
		if participantCount != 2 {
			t.Errorf("Expected 2 participants after leave, got %d", participantCount)
		}

		var user3Participant models.ChannelParticipant
		err := db.Postgresql.Where("channel_id = ? AND user_id = ?", groupDMChannelID, user3.ID).First(&user3Participant).Error
		if err == nil {
			t.Error("User3 should no longer be a participant")
		}

		t.Logf("✅ User successfully left group DM channel")
	})

	t.Run("Fail to Leave - User Not a Participant", func(t *testing.T) {
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

		for _, user := range []models.User{user1, user2} {
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
		req.Header.Set("Authorization", "Bearer "+token4)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 403, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		message, ok := response["message"].(string)
		if !ok {
			t.Fatal("Response missing message field")
		}

		if message != "user is not a participant in the group DM channel" {
			t.Errorf("Expected forbidden message, got: %s", message)
		}

		t.Logf("✅ Correctly prevented non-participant from leaving")
	})

	t.Run("Fail to Leave - Channel Does Not Exist", func(t *testing.T) {
		nonExistentChannelID := utility.GenerateUUID()

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.DELETE("/api/v1/organisations/:org_id/group-dms/:channel_id/leave", middleware.Authorize(db.Postgresql), controller.LeaveGroupDMChannel)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/leave", org.ID, nonExistentChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
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

	t.Run("Last Participant Leaves - Channel Gets Deleted", func(t *testing.T) {
		groupDMChannelID := utility.GenerateUUID()
		participantHash := utility.GenerateUUID()

		groupDMChannel := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          user3.ID,
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
			UserId:    user3.ID,
			OrgId:     org.ID,
		}
		if err := db.Postgresql.Create(&participant).Error; err != nil {
			t.Fatalf("Failed to create channel participant: %v", err)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.DELETE("/api/v1/organisations/:org_id/group-dms/:channel_id/leave", middleware.Authorize(db.Postgresql), controller.LeaveGroupDMChannel)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/leave", org.ID, groupDMChannelID), nil)
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var channelCheck models.DmChannels
		err := db.Postgresql.Where("channel_id = ?", groupDMChannelID).First(&channelCheck).Error
		if err == nil {
			t.Error("Channel should have been deleted when last participant left")
		}

		var participantCount int64
		db.Postgresql.Model(&models.ChannelParticipant{}).Where("channel_id = ?", groupDMChannelID).Count(&participantCount)
		if participantCount != 0 {
			t.Errorf("Expected 0 participants after channel deletion, got %d", participantCount)
		}

		t.Logf("✅ Channel correctly deleted when last participant left")
	})

	t.Run("Fail to Leave - Invalid Channel ID Format", func(t *testing.T) {
		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.DELETE("/api/v1/organisations/:org_id/group-dms/:channel_id/leave", middleware.Authorize(db.Postgresql), controller.LeaveGroupDMChannel)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/organisations/%s/group-dms/%s/leave", org.ID, "invalid-uuid"), nil)
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Logf("✅ Correctly rejected invalid channel ID format")
	})
}
