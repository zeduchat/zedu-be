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

func TestAddParticipantsToGroupDM(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_addpart_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserOne",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_addpart_user1_%v", currUUID),
	}

	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_addpart_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserTwo",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_addpart_user2_%v", currUUID),
	}

	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_addpart_user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserThree",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_addpart_user3_%v", currUUID),
	}

	user4SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_addpart_user4_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserFour",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_addpart_user4_%v", currUUID),
	}

	user5SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("groupdm_addpart_user5_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDM",
		LastName:    "UserFive",
		Password:    "password",
		UserName:    fmt.Sprintf("groupdm_addpart_user5_%v", currUUID),
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
	tst.SignupUser(t, gin.Default(), authController, user5SignUpData, false)
	token1 := tst.GetLoginToken(t, gin.Default(), authController, loginData1)

	var user1, user2, user3, user4, user5 models.User
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
	if err := db.Postgresql.Where("email = ?", user5SignUpData.Email).First(&user5).Error; err != nil {
		t.Fatalf("Failed to get user5: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	t.Run("Successfully Add Multiple Participants to Group DM", func(t *testing.T) {
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
		r.POST("/api/v1/organisations/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		requestBody := map[string]interface{}{
			"user_ids": []string{user3.ID, user4.ID, user5.ID},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/participants", groupDMChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

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

		addedCount, ok := data["added_count"].(float64)
		if !ok || int(addedCount) != 3 {
			t.Errorf("Expected added_count to be 3, got %v", data["added_count"])
		}

		skippedCount, ok := data["skipped_count"].(float64)
		if !ok || int(skippedCount) != 0 {
			t.Errorf("Expected skipped_count to be 0, got %v", data["skipped_count"])
		}

		participants, ok := data["participants"].([]interface{})
		if !ok {
			t.Fatal("participants field is missing or not an array")
		}

		if len(participants) != 5 {
			t.Errorf("Expected 5 participants after adding, got %d", len(participants))
		}

		var participantCount int64
		db.Postgresql.Model(&models.ChannelParticipant{}).Where("channel_id = ?", groupDMChannelID).Count(&participantCount)
		if participantCount != 5 {
			t.Errorf("Expected 5 participants in database, got %d", participantCount)
		}

		t.Logf("✅ Successfully added %d participants to group DM", int(addedCount))
	})

	t.Run("Fail to Add - Channel at Maximum Capacity (10)", func(t *testing.T) {
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

		users := []models.User{}
		for i := 0; i < 10; i++ {
			userSignUpData := models.CreateUserRequestModel{
				Email:       fmt.Sprintf("groupdm_capacity_user%d_%v@qa.team", i, currUUID),
				PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
				FirstName:   "Capacity",
				LastName:    fmt.Sprintf("User%d", i),
				Password:    "password",
				UserName:    fmt.Sprintf("capacity_user%d_%v", i, currUUID),
			}
			tst.SignupUser(t, gin.Default(), authController, userSignUpData, false)

			var user models.User
			if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
				t.Fatalf("Failed to get user: %v", err)
			}
			users = append(users, user)

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
		r.POST("/api/v1/organisations/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		requestBody := map[string]interface{}{
			"user_ids": []string{user1.ID},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/participants", groupDMChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Logf("✅ Correctly prevented adding participants when at maximum capacity")
	})

	t.Run("Fail to Add - Some Users Already in Channel (Should Skip Duplicates)", func(t *testing.T) {
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
		r.POST("/api/v1/organisations/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		requestBody := map[string]interface{}{
			"user_ids": []string{user2.ID, user3.ID},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/participants", groupDMChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token1)
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

		if message != "no new participants were added (all users are already participants or invalid)" {
			t.Errorf("Expected duplicate message, got: %s", message)
		}

		t.Logf("✅ Correctly handled duplicate participants")
	})

	t.Run("Fail to Add - Channel Does Not Exist", func(t *testing.T) {
		nonExistentChannelID := utility.GenerateUUID()

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		requestBody := map[string]interface{}{
			"user_ids": []string{user4.ID},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/participants", nonExistentChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

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

	t.Run("Fail to Add - Invalid User IDs", func(t *testing.T) {
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
		r.POST("/api/v1/organisations/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		requestBody := map[string]interface{}{
			"user_ids": []string{"invalid-uuid"},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/participants", groupDMChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token1)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Logf("✅ Correctly rejected invalid user IDs")
	})

	t.Run("Fail to Add - Adding Would Exceed 10 Participant Limit", func(t *testing.T) {
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

		users := []models.User{}
		for i := 0; i < 8; i++ {
			userSignUpData := models.CreateUserRequestModel{
				Email:       fmt.Sprintf("groupdm_exceed_user%d_%v@qa.team", i, currUUID),
				PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
				FirstName:   "Exceed",
				LastName:    fmt.Sprintf("User%d", i),
				Password:    "password",
				UserName:    fmt.Sprintf("exceed_user%d_%v", i, currUUID),
			}
			tst.SignupUser(t, gin.Default(), authController, userSignUpData, false)

			var user models.User
			if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
				t.Fatalf("Failed to get user: %v", err)
			}
			users = append(users, user)

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

		newUsers := []models.User{}
		for i := 0; i < 3; i++ {
			userSignUpData := models.CreateUserRequestModel{
				Email:       fmt.Sprintf("groupdm_new_user%d_%v@qa.team", i, currUUID),
				PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
				FirstName:   "New",
				LastName:    fmt.Sprintf("User%d", i),
				Password:    "password",
				UserName:    fmt.Sprintf("new_user%d_%v", i, currUUID),
			}
			tst.SignupUser(t, gin.Default(), authController, userSignUpData, false)

			var user models.User
			if err := db.Postgresql.Where("email = ?", userSignUpData.Email).First(&user).Error; err != nil {
				t.Fatalf("Failed to get user: %v", err)
			}
			newUsers = append(newUsers, user)
		}

		extReq := request.ExternalRequest{Logger: logger, Test: true}
		controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

		r := gin.Default()
		r.POST("/api/v1/organisations/group-dms/:channel_id/participants", middleware.Authorize(db.Postgresql), controller.AddParticipantsToGroupDMChannel)

		requestBody := map[string]interface{}{
			"user_ids": []string{newUsers[0].ID, newUsers[1].ID, newUsers[2].ID},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/group-dms/%s/participants", groupDMChannelID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token1)
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

		if message != "adding 3 participants would exceed maximum capacity of 10 (current: 8)" {
			t.Errorf("Expected capacity exceeded message, got: %s", message)
		}

		t.Logf("✅ Correctly prevented exceeding 10 participant limit")
	})
}
