package test_buzz

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
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func setupDirectCallUsers(t *testing.T, db *storage.Database, logger *utility.Logger, validatorRef *validator.Validate, count int) ([]models.User, []string, string) {
	currUUID := utility.GenerateUUID()
	users := make([]models.User, count)
	tokens := make([]string, count)

	for i := 0; i < count; i++ {
		signUpData := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("calluser%d_%v@qa.team", i, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   fmt.Sprintf("CallUser%d", i),
			LastName:    "Test",
			Password:    "password",
			UserName:    fmt.Sprintf("calluser%d_%v", i, currUUID),
		}
		loginData := models.LoginRequestModel{
			Email:    signUpData.Email,
			Password: signUpData.Password,
		}

		authCtrl := auth.Controller{
			Db:        db,
			Validator: validatorRef,
			Logger:    logger,
			ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
		}

		r := gin.Default()
		tst.SignupUser(t, r, authCtrl, signUpData, false)
		tokens[i] = tst.GetLoginToken(t, r, authCtrl, loginData)

		if err := db.Postgresql.Where("email = ?", signUpData.Email).First(&users[i]).Error; err != nil {
			t.Fatalf("failed to fetch user%d: %v", i, err)
		}
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", users[0].ID).First(&org).Error; err != nil {
		t.Fatalf("failed to get organization for user0: %v", err)
	}

	return users, tokens, org.ID
}

func createTestDMChannel(t *testing.T, db *storage.Database, user1, user2 models.User, orgID string) string {
	dmChannelID := utility.GenerateUUID()
	participantID := user2.ID

	dm1 := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        user1.ID,
		ChannelId:     dmChannelID,
		OrgId:         orgID,
		ParticipantId: &participantID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	if err := db.Postgresql.Create(&dm1).Error; err != nil {
		t.Fatalf("failed to create DM channel (user1): %v", err)
	}

	dm2 := models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        user2.ID,
		ChannelId:     dmChannelID,
		OrgId:         orgID,
		ParticipantId: &user1.ID,
		ChatType:      "user",
		ChannelType:   "dm",
	}
	if err := db.Postgresql.Create(&dm2).Error; err != nil {
		t.Fatalf("failed to create DM channel (user2): %v", err)
	}

	return dmChannelID
}

func createTestGroupDMChannel(t *testing.T, db *storage.Database, users []models.User, orgID string) string {
	groupDMChannelID := utility.GenerateUUID()
	participantHash := utility.GenerateUUID()

	for _, u := range users {
		participant := models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupDMChannelID,
			UserId:    u.ID,
			OrgId:     orgID,
		}
		if err := db.Postgresql.Create(&participant).Error; err != nil {
			t.Fatalf("failed to create channel participant: %v", err)
		}

		groupDM := models.DmChannels{
			ID:              utility.GenerateUUID(),
			UserId:          u.ID,
			ChannelId:       groupDMChannelID,
			OrgId:           orgID,
			ParticipantHash: participantHash,
			ChatType:        "user",
			ChannelType:     "group_dm",
		}
		if err := db.Postgresql.Create(&groupDM).Error; err != nil {
			t.Fatalf("failed to create group DM entry: %v", err)
		}
	}

	return groupDMChannelID
}

func setupDirectCallRouter(buzzCtrl buzz.Controller, db *storage.Database) *gin.Engine {
	r := gin.Default()
	buzzURL := r.Group("/api/v1/buzz", middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
	{
		buzzURL.POST("/direct-call", buzzCtrl.InitiateDirectCall)
		buzzURL.POST("/:id/respond", buzzCtrl.RespondToCall)
	}
	return r
}

func TestDirectCallInitiate_DM_Success(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	users, tokens, orgID := setupDirectCallUsers(t, db, logger, validatorRef, 2)
	dmChannelID := createTestDMChannel(t, db, users[0], users[1], orgID)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	reqBody := models.InitiateDirectCallRequest{ChannelID: dmChannelID}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens[0])

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Response Status: %d", rr.Code)
	t.Logf("Response Body: %s", rr.Body.String())

	tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing data field")
	}

	if data["buzz_id"] == nil {
		t.Error("expected buzz_id in response")
	}
	if data["channel_id"] != dmChannelID {
		t.Errorf("expected channel_id %s, got %v", dmChannelID, data["channel_id"])
	}

	participants, ok := data["participants"].([]interface{})
	if !ok || len(participants) == 0 {
		t.Fatal("expected non-empty participants in response")
	}

	callerAccepted := false
	otherPending := false
	for _, p := range participants {
		part := p.(map[string]interface{})
		if part["user_id"] == users[0].ID && part["status"] == models.CallStatusAccepted {
			callerAccepted = true
		}
		if part["user_id"] == users[1].ID && part["status"] == models.CallStatusPending {
			otherPending = true
		}
	}

	if !callerAccepted {
		t.Error("expected caller to have status 'accepted'")
	}
	if !otherPending {
		t.Error("expected other participant to have status 'pending'")
	}

	t.Logf("✅ Direct call in DM initiated successfully with correct participant statuses")
}

func TestDirectCallInitiate_GroupDM_Success(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	users, tokens, orgID := setupDirectCallUsers(t, db, logger, validatorRef, 3)
	groupDMChannelID := createTestGroupDMChannel(t, db, users, orgID)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	reqBody := models.InitiateDirectCallRequest{ChannelID: groupDMChannelID}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens[0])

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Response Status: %d", rr.Code)
	t.Logf("Response Body: %s", rr.Body.String())

	tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	data := resp["data"].(map[string]interface{})
	participants := data["participants"].([]interface{})

	if len(participants) != 3 {
		t.Errorf("expected 3 participants, got %d", len(participants))
	}

	for _, p := range participants {
		part := p.(map[string]interface{})
		if part["user_id"] == users[0].ID {
			if part["status"] != models.CallStatusAccepted {
				t.Errorf("caller should have status 'accepted', got %v", part["status"])
			}
		} else {
			if part["status"] != models.CallStatusPending {
				t.Errorf("others should have status 'pending', got %v", part["status"])
			}
		}
	}

	t.Logf("✅ Direct call in Group DM initiated successfully with %d participants", len(participants))
}

func TestDirectCallInitiate_Forbidden_NotInChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	users, tokens, orgID := setupDirectCallUsers(t, db, logger, validatorRef, 3)
	dmChannelID := createTestDMChannel(t, db, users[0], users[1], orgID)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	// User2 (index 2) is NOT in the DM channel
	reqBody := models.InitiateDirectCallRequest{ChannelID: dmChannelID}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens[2])

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Response Status: %d, Body: %s", rr.Code, rr.Body.String())

	if rr.Code != http.StatusForbidden && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 403 or 400, got %d", rr.Code)
	}
	t.Logf("✅ Correctly rejected non-member from initiating direct call")
}

func TestDirectCallInitiate_BadRequest_NonDMChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	_, tokens, _ := setupDirectCallUsers(t, db, logger, validatorRef, 1)

	// Use a random UUID that isn't a DM channel
	reqBody := models.InitiateDirectCallRequest{ChannelID: utility.GenerateUUID()}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens[0])

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Response Status: %d", rr.Code)
	if rr.Code != http.StatusNotFound && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 404 or 400 for non-DM channel, got %d", rr.Code)
	}
	t.Logf("✅ Correctly rejected direct call on non-DM channel")
}

func TestDirectCallRespond_Accept(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	users, tokens, orgID := setupDirectCallUsers(t, db, logger, validatorRef, 2)
	dmChannelID := createTestDMChannel(t, db, users[0], users[1], orgID)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	// Initiate call as user 0
	initReq := models.InitiateDirectCallRequest{ChannelID: dmChannelID}
	var initBody bytes.Buffer
	json.NewEncoder(&initBody).Encode(initReq)

	initHttpReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &initBody)
	initHttpReq.Header.Set("Content-Type", "application/json")
	initHttpReq.Header.Set("Authorization", "Bearer "+tokens[0])

	initRR := httptest.NewRecorder()
	r.ServeHTTP(initRR, initHttpReq)

	if initRR.Code != http.StatusCreated {
		t.Fatalf("failed to initiate call, got %d: %s", initRR.Code, initRR.Body.String())
	}

	var initResp map[string]interface{}
	json.Unmarshal(initRR.Body.Bytes(), &initResp)
	buzzID := initResp["data"].(map[string]interface{})["buzz_id"].(string)

	// User 1 accepts the call
	respondReq := models.RespondToCallRequest{Action: "accept"}
	var respondBody bytes.Buffer
	json.NewEncoder(&respondBody).Encode(respondReq)

	respondHttpReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/respond", &respondBody)
	respondHttpReq.Header.Set("Content-Type", "application/json")
	respondHttpReq.Header.Set("Authorization", "Bearer "+tokens[1])

	respondRR := httptest.NewRecorder()
	r.ServeHTTP(respondRR, respondHttpReq)

	t.Logf("Respond Status: %d, Body: %s", respondRR.Code, respondRR.Body.String())
	tst.AssertStatusCode(t, respondRR.Code, http.StatusOK)

	var respondResp map[string]interface{}
	json.Unmarshal(respondRR.Body.Bytes(), &respondResp)
	data := respondResp["data"].(map[string]interface{})

	participants := data["participants"].([]interface{})
	allAccepted := false
	for _, p := range participants {
		part := p.(map[string]interface{})
		if part["user_id"] == users[1].ID && part["status"] == models.CallStatusAccepted {
			allAccepted = true
		}
	}
	if !allAccepted {
		t.Error("expected user1 participant status to be 'accepted' after accepting call")
	}

	t.Logf("✅ User successfully accepted direct call, got agora_token: %v", data["agora_token"] != nil)
}

func TestDirectCallRespond_Decline(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	users, tokens, orgID := setupDirectCallUsers(t, db, logger, validatorRef, 2)
	dmChannelID := createTestDMChannel(t, db, users[0], users[1], orgID)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	// Initiate call
	initReq := models.InitiateDirectCallRequest{ChannelID: dmChannelID}
	var initBody bytes.Buffer
	json.NewEncoder(&initBody).Encode(initReq)

	initHttpReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &initBody)
	initHttpReq.Header.Set("Content-Type", "application/json")
	initHttpReq.Header.Set("Authorization", "Bearer "+tokens[0])

	initRR := httptest.NewRecorder()
	r.ServeHTTP(initRR, initHttpReq)

	if initRR.Code != http.StatusCreated {
		t.Fatalf("failed to initiate call: %s", initRR.Body.String())
	}

	var initResp map[string]interface{}
	json.Unmarshal(initRR.Body.Bytes(), &initResp)
	buzzID := initResp["data"].(map[string]interface{})["buzz_id"].(string)

	// User 1 declines
	respondReq := models.RespondToCallRequest{Action: "decline"}
	var respondBody bytes.Buffer
	json.NewEncoder(&respondBody).Encode(respondReq)

	respondHttpReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/respond", &respondBody)
	respondHttpReq.Header.Set("Content-Type", "application/json")
	respondHttpReq.Header.Set("Authorization", "Bearer "+tokens[1])

	respondRR := httptest.NewRecorder()
	r.ServeHTTP(respondRR, respondHttpReq)

	t.Logf("Decline Status: %d, Body: %s", respondRR.Code, respondRR.Body.String())
	tst.AssertStatusCode(t, respondRR.Code, http.StatusOK)

	var resp map[string]interface{}
	json.Unmarshal(respondRR.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	participants := data["participants"].([]interface{})

	declined := false
	for _, p := range participants {
		part := p.(map[string]interface{})
		if part["user_id"] == users[1].ID && part["status"] == models.CallStatusDeclined {
			declined = true
		}
	}
	if !declined {
		t.Error("expected user1 to have status 'declined'")
	}

	t.Logf("✅ User successfully declined direct call")
}

func TestDirectCallRespond_Timeout(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	users, tokens, orgID := setupDirectCallUsers(t, db, logger, validatorRef, 2)
	dmChannelID := createTestDMChannel(t, db, users[0], users[1], orgID)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	// Initiate call
	initReq := models.InitiateDirectCallRequest{ChannelID: dmChannelID}
	var initBody bytes.Buffer
	json.NewEncoder(&initBody).Encode(initReq)

	initHttpReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/direct-call", &initBody)
	initHttpReq.Header.Set("Content-Type", "application/json")
	initHttpReq.Header.Set("Authorization", "Bearer "+tokens[0])

	initRR := httptest.NewRecorder()
	r.ServeHTTP(initRR, initHttpReq)

	if initRR.Code != http.StatusCreated {
		t.Fatalf("failed to initiate call: %s", initRR.Body.String())
	}

	var initResp map[string]interface{}
	json.Unmarshal(initRR.Body.Bytes(), &initResp)
	buzzID := initResp["data"].(map[string]interface{})["buzz_id"].(string)

	// Simulate timeout by backdating created_at to 6 minutes ago
	pastTime := time.Now().UTC().Add(-6 * time.Minute)
	if err := db.Postgresql.Exec("UPDATE buzzs SET created_at = ? WHERE id = ?", pastTime, buzzID).Error; err != nil {
		t.Fatalf("failed to backdate buzz created_at: %v", err)
	}

	// User 1 sends any action — should be overridden to timeout
	respondReq := models.RespondToCallRequest{Action: "accept"}
	var respondBody bytes.Buffer
	json.NewEncoder(&respondBody).Encode(respondReq)

	respondHttpReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/respond", &respondBody)
	respondHttpReq.Header.Set("Content-Type", "application/json")
	respondHttpReq.Header.Set("Authorization", "Bearer "+tokens[1])

	respondRR := httptest.NewRecorder()
	r.ServeHTTP(respondRR, respondHttpReq)

	t.Logf("Timeout Status: %d, Body: %s", respondRR.Code, respondRR.Body.String())
	tst.AssertStatusCode(t, respondRR.Code, http.StatusOK)

	var resp map[string]interface{}
	json.Unmarshal(respondRR.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	participants := data["participants"].([]interface{})

	timedOut := false
	for _, p := range participants {
		part := p.(map[string]interface{})
		if part["user_id"] == users[1].ID && part["status"] == models.CallStatusTimeout {
			timedOut = true
		}
	}
	if !timedOut {
		t.Error("expected user1 to have status 'timeout' after 5-min ringing period expired")
	}

	t.Logf("✅ Correctly enforced 5-minute ringing timeout")
}

func TestDirectCallRespond_InvalidAction(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	_, tokens, _ := setupDirectCallUsers(t, db, logger, validatorRef, 1)

	buzzCtrl := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := setupDirectCallRouter(buzzCtrl, db)

	respondReq := map[string]string{"action": "invalid_action"}
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(respondReq)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+utility.GenerateUUID()+"/respond", &body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokens[0])

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	t.Logf("Invalid Action Status: %d", rr.Code)
	if rr.Code != http.StatusUnprocessableEntity && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 422 or 400 for invalid action, got %d", rr.Code)
	}
	t.Logf("✅ Correctly rejected invalid action value")
}
