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

func TestUpsertGroupDescription(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	// Create user 1 (will be admin/creator)
	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("grpdesc_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDesc",
		LastName:    "User1",
		Password:    "password",
		UserName:    fmt.Sprintf("grpdesc_user1_%v", currUUID),
	}

	// Create user 2 (participant)
	user2SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("grpdesc_user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDesc",
		LastName:    "User2",
		Password:    "password",
		UserName:    fmt.Sprintf("grpdesc_user2_%v", currUUID),
	}

	// Create user 3 (non-participant for testing unauthorized access)
	user3SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("grpdesc_user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GroupDesc",
		LastName:    "User3",
		Password:    "password",
		UserName:    fmt.Sprintf("grpdesc_user3_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	loginData3 := models.LoginRequestModel{
		Email:    user3SignUpData.Email,
		Password: user3SignUpData.Password,
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
	token1 := tst.GetLoginToken(t, r, authController, loginData1)
	token3 := tst.GetLoginToken(t, gin.Default(), authController, loginData3)

	var user1, user2 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	if err := db.Postgresql.Where("email = ?", user2SignUpData.Email).First(&user2).Error; err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Create a group DM channel
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

	// Create channel participants for user1 and user2
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

	// Create a regular DM channel (for testing that group description cannot be set on DMs)
	dmChannelID := utility.GenerateUUID()
	participantID := user2.ID
	dmChannel := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        user1.ID,
		ChannelId:     dmChannelID,
		OrgId:         org.ID,
		ParticipantId: &participantID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
		t.Fatalf("Failed to create DM channel: %v", err)
	}

	extReq := request.ExternalRequest{Logger: logger, Test: true}
	controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: extReq}

	t.Run("Successfully set group description", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "This is our team discussion group",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, groupDMChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] != "Group description updated successfully" {
			t.Errorf("Expected success message, got: %v", response["message"])
		}

		// Verify the description was saved
		var updatedChannel models.DmChannels
		if err := db.Postgresql.Where("channel_id = ?", groupDMChannelID).First(&updatedChannel).Error; err != nil {
			t.Fatalf("Failed to fetch updated channel: %v", err)
		}

		if updatedChannel.GroupDescription != "This is our team discussion group" {
			t.Errorf("Expected description 'This is our team discussion group', got '%s'", updatedChannel.GroupDescription)
		}

		t.Log("✅ Successfully set group description")
	})

	t.Run("Successfully update existing group description", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "Updated group description",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, groupDMChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		// Verify the description was updated
		var updatedChannel models.DmChannels
		if err := db.Postgresql.Where("channel_id = ?", groupDMChannelID).First(&updatedChannel).Error; err != nil {
			t.Fatalf("Failed to fetch updated channel: %v", err)
		}

		if updatedChannel.GroupDescription != "Updated group description" {
			t.Errorf("Expected description 'Updated group description', got '%s'", updatedChannel.GroupDescription)
		}

		t.Log("✅ Successfully updated existing group description")
	})

	t.Run("Fail when trying to set description on regular DM", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "This should fail",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, dmChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] != "group description can only be set for group DM channels" {
			t.Logf("Response message: %v", response["message"])
		}

		t.Log("✅ Correctly rejected setting description on regular DM")
	})

	t.Run("Fail when non-participant tries to set description", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "This should fail - not a participant",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, groupDMChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token3)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Log("✅ Correctly rejected non-participant trying to set description")
	})

	t.Run("Fail when channel does not exist", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		nonExistentChannelID := utility.GenerateUUID()
		reqBody := map[string]string{
			"description": "This should fail - channel doesn't exist",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, nonExistentChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Log("✅ Correctly rejected setting description on non-existent channel")
	})

	t.Run("Fail with empty description", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, groupDMChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Log("✅ Correctly rejected empty description")
	})

	t.Run("Fail with invalid org_id format", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "Test description",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", "invalid-uuid", groupDMChannelID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Log("✅ Correctly rejected invalid org_id format")
	})

	t.Run("Fail with invalid channel_id format", func(t *testing.T) {
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)

		reqBody := map[string]string{
			"description": "Test description",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, "invalid-uuid"), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rr.Code, rr.Body.String())
		}

		t.Log("✅ Correctly rejected invalid channel_id format")
	})

	t.Run("Verify group description is returned in GetDmParticipants", func(t *testing.T) {
		// First, set a group description
		r := gin.Default()
		r.PUT("/api/v1/organisations/:org_id/dms/:channel_id/description", middleware.Authorize(db.Postgresql), controller.UpsertGroupDescription)
		r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id", middleware.Authorize(db.Postgresql), controller.GetDmParticipants)

		expectedDescription := "Team discussion for project alpha"
		reqBody := map[string]string{
			"description": expectedDescription,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		putReq, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/dms/%s/description", org.ID, groupDMChannelID), bytes.NewBuffer(bodyBytes))
		putReq.Header.Set("Content-Type", "application/json")
		putReq.Header.Set("Authorization", "Bearer "+token1)

		putRr := httptest.NewRecorder()
		r.ServeHTTP(putRr, putReq)

		if putRr.Code != http.StatusOK {
			t.Fatalf("Failed to set group description: %d - %s", putRr.Code, putRr.Body.String())
		}

		// Now fetch participants and verify group_description is returned
		getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", org.ID, groupDMChannelID), nil)
		getReq.Header.Set("Authorization", "Bearer "+token1)

		getRr := httptest.NewRecorder()
		r.ServeHTTP(getRr, getReq)

		if getRr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", getRr.Code, getRr.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(getRr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected data field in response, got: %v", response)
		}

		groupDescription, exists := data["group_description"].(string)
		if !exists {
			t.Errorf("group_description field missing from response")
		}

		if groupDescription != expectedDescription {
			t.Errorf("Expected group_description '%s', got '%s'", expectedDescription, groupDescription)
		}

		responseType, _ := data["type"].(string)
		if responseType != "groupdm" {
			t.Errorf("Expected type 'groupdm', got '%s'", responseType)
		}

		t.Logf("✅ group_description correctly returned: '%s'", groupDescription)
	})
}
