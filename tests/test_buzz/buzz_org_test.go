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
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateOrgBuzz(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	hostEmail := utility.GenerateUUID() + "@qa.team"
	hostSignUp := models.CreateUserRequestModel{
		Email:       hostEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "HostUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("hostuser_%v", utility.GenerateUUID())}

	hostLogin := models.LoginRequestModel{Email: hostSignUp.Email, Password: hostSignUp.Password}

	tst.SignupUser(t, gin.Default(), auth, hostSignUp, false)
	hostToken := tst.GetLoginToken(t, gin.Default(), auth, hostLogin)
	if hostToken == "" {
		t.Fatalf("failed to obtain host login token")
	}

	router, _ := SetupBuzzEndTestRouter(logger, validatorRef)
	var hostUser models.User
	if err := db.Postgresql.Where("email = ?", hostSignUp.Email).First(&hostUser).Error; err != nil {
		t.Fatalf("failed to fetch host user: %v", err)
	}

	t.Run("CreateOrgBuzzSuccessfully", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "test_" + utility.GenerateUUID(),
			OrganisationID: hostUser.CurrentOrg.String(),
			OwnerId:        hostUser.ID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&channel).Error; err != nil {
			t.Fatalf("Failed to create test channel: %v", err)
		}

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     hostUser.ID,
			Username:   hostSignUp.UserName,
			CreatedAt:  time.Now(),
		}
		if err := db.Postgresql.Create(&userChannel).Error; err != nil {
			t.Logf("Warning: Failed to add user to channel: %v", err)
		}

		createBuzzData := models.CreateBuzzRequest{}

		jsonData, _ := json.Marshal(createBuzzData)
		url := "/api/v1/buzz/org/create"
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)

		if dataM["buzz_id"] == nil || dataM["buzz_id"].(string) == "" {
			t.Error("expected buzz_id to be set")
		}

		if dataM["host_id"].(string) != hostUser.ID {
			t.Errorf("expected host_id %s, got %s", hostUser.ID, dataM["host_id"].(string))
		}

		buzzID := dataM["buzz_id"].(string)
		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("failed to fetch buzz from database: %v", err)
		}

		if buzz.OrgID == nil {
			t.Error("expected org_id to be set")
		} else if *buzz.OrgID != hostUser.CurrentOrg.String() {
			t.Errorf("expected org_id %s, got %s", hostUser.CurrentOrg.String(), *buzz.OrgID)
		}

		if buzz.HostID != hostUser.ID {
			t.Errorf("expected host_id %s, got %s", hostUser.ID, buzz.HostID)
		}

		if buzz.BuzzType != models.BuzzTypeOrganization {
			t.Errorf("expected buzz_type %s, got %s", models.BuzzTypeOrganization, buzz.BuzzType)
		}

		if buzz.ChannelID != "00000000-0000-0000-0000-000000000000" {
			t.Errorf("expected channel_id to be 00000000-0000-0000-0000-000000000000 for org buzz, got %s", buzz.ChannelID)
		}
	})

	t.Run("CreateOrgBuzzFailsWithoutOrgContext", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "test_" + utility.GenerateUUID(),
			OrganisationID: hostUser.CurrentOrg.String(),
			OwnerId:        hostUser.ID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&channel).Error; err != nil {
			t.Fatalf("Failed to create test channel: %v", err)
		}

		createBuzzData := models.CreateBuzzRequest{}

		jsonData, _ := json.Marshal(createBuzzData)
		url := "/api/v1/buzz/org/create"
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})
}

func TestGetOrgBuzzList(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	hostEmail := utility.GenerateUUID() + "@qa.team"
	hostSignUp := models.CreateUserRequestModel{
		Email:       hostEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "HostUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("hostuser_%v", utility.GenerateUUID())}

	hostLogin := models.LoginRequestModel{Email: hostSignUp.Email, Password: hostSignUp.Password}

	tst.SignupUser(t, gin.Default(), auth, hostSignUp, false)
	hostToken := tst.GetLoginToken(t, gin.Default(), auth, hostLogin)
	if hostToken == "" {
		t.Fatalf("failed to obtain host login token")
	}

	router, _ := SetupBuzzEndTestRouter(logger, validatorRef)
	var hostUser models.User
	if err := db.Postgresql.Where("email = ?", hostSignUp.Email).First(&hostUser).Error; err != nil {
		t.Fatalf("failed to fetch host user: %v", err)
	}

	t.Run("GetOrgBuzzListSuccessfully", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "test_" + utility.GenerateUUID(),
			OrganisationID: hostUser.CurrentOrg.String(),
			OwnerId:        hostUser.ID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&channel).Error; err != nil {
			t.Fatalf("Failed to create test channel: %v", err)
		}

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     hostUser.ID,
			Username:   hostSignUp.UserName,
			CreatedAt:  time.Now(),
		}
		if err := db.Postgresql.Create(&userChannel).Error; err != nil {
			t.Logf("Warning: Failed to add user to channel: %v", err)
		}

		createBuzzData := models.CreateBuzzRequest{}

		jsonData, _ := json.Marshal(createBuzzData)
		createURL := "/api/v1/buzz/org/create"
		createReq, err := http.NewRequest(http.MethodPost, createURL, bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatal(err)
		}

		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+hostToken)

		createRR := httptest.NewRecorder()
		router.ServeHTTP(createRR, createReq)

		if createRR.Code != http.StatusCreated {
			t.Fatalf("failed to create org buzz: %d", createRR.Code)
		}

		listURL := "/api/v1/buzz/org"
		listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		listReq.Header.Set("Authorization", "Bearer "+hostToken)

		listRR := httptest.NewRecorder()
		router.ServeHTTP(listRR, listReq)

		tst.AssertStatusCode(t, listRR.Code, http.StatusOK)

		data := tst.ParseResponse(listRR)
		dataM := data["data"].(map[string]any)

		buzzes := dataM["buzzes"].([]any)
		if len(buzzes) == 0 {
			t.Error("expected at least one buzz in the list")
		}

		total := int(dataM["total"].(float64))
		if total == 0 {
			t.Error("expected total to be greater than 0")
		}

		firstBuzz := buzzes[0].(map[string]any)
		if firstBuzz["org_id"].(string) != hostUser.CurrentOrg.String() {
			t.Errorf("expected org_id %s, got %s", hostUser.CurrentOrg.String(), firstBuzz["org_id"].(string))
		}

		if firstBuzz["host_id"].(string) != hostUser.ID {
			t.Errorf("expected host_id %s, got %s", hostUser.ID, firstBuzz["host_id"].(string))
		}
	})

	t.Run("GetOrgBuzzListFailsWithoutAuth", func(t *testing.T) {
		listURL := "/api/v1/buzz/org"
		listReq, err := http.NewRequest(http.MethodGet, listURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		listRR := httptest.NewRecorder()
		router.ServeHTTP(listRR, listReq)

		tst.AssertStatusCode(t, listRR.Code, http.StatusUnauthorized)
	})
}

func TestExistingRoutesWithOrgID(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	hostEmail := utility.GenerateUUID() + "@qa.team"
	hostSignUp := models.CreateUserRequestModel{
		Email:       hostEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "HostUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("hostuser_%v", utility.GenerateUUID())}

	hostLogin := models.LoginRequestModel{Email: hostSignUp.Email, Password: hostSignUp.Password}

	tst.SignupUser(t, gin.Default(), auth, hostSignUp, false)
	hostToken := tst.GetLoginToken(t, gin.Default(), auth, hostLogin)
	if hostToken == "" {
		t.Fatalf("failed to obtain host login token")
	}

	router, _ := SetupBuzzEndTestRouter(logger, validatorRef)
	var hostUser models.User
	if err := db.Postgresql.Where("email = ?", hostSignUp.Email).First(&hostUser).Error; err != nil {
		t.Fatalf("failed to fetch host user: %v", err)
	}

	t.Run("JoinOrgBuzzSuccessfully", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "test_" + utility.GenerateUUID(),
			OrganisationID: hostUser.CurrentOrg.String(),
			OwnerId:        hostUser.ID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&channel).Error; err != nil {
			t.Fatalf("Failed to create test channel: %v", err)
		}

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     hostUser.ID,
			Username:   hostSignUp.UserName,
			CreatedAt:  time.Now(),
		}
		if err := db.Postgresql.Create(&userChannel).Error; err != nil {
			t.Logf("Warning: Failed to add user to channel: %v", err)
		}

		createBuzzData := models.CreateBuzzRequest{}

		jsonData, _ := json.Marshal(createBuzzData)
		createURL := "/api/v1/buzz/org/create"
		createReq, err := http.NewRequest(http.MethodPost, createURL, bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatal(err)
		}

		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+hostToken)

		createRR := httptest.NewRecorder()
		router.ServeHTTP(createRR, createReq)

		if createRR.Code != http.StatusCreated {
			t.Fatalf("failed to create org buzz: %d", createRR.Code)
		}

		createData := tst.ParseResponse(createRR)
		createDataM := createData["data"].(map[string]any)
		buzzID := createDataM["buzz_id"].(string)

		participantEmail := utility.GenerateUUID() + "@qa.team"
		participantSignUp := models.CreateUserRequestModel{
			Email:    participantEmail,
			Password: "password",
			UserName: fmt.Sprintf("participant_%v", utility.GenerateUUID()),
		}
		participantLogin := models.LoginRequestModel{Email: participantSignUp.Email, Password: participantSignUp.Password}

		tst.SignupUser(t, gin.Default(), auth, participantSignUp, false)
		participantToken := tst.GetLoginToken(t, gin.Default(), auth, participantLogin)
		if participantToken == "" {
			t.Fatalf("failed to obtain participant login token")
		}

		var participantUser models.User
		if err := db.Postgresql.Where("email = ?", participantSignUp.Email).First(&participantUser).Error; err != nil {
			t.Fatalf("failed to fetch participant user: %v", err)
		}

		participantUser.CurrentOrg = hostUser.CurrentOrg
		if err := db.Postgresql.Save(&participantUser).Error; err != nil {
			t.Fatalf("failed to update participant org: %v", err)
		}

		participantChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     participantUser.ID,
			Username:   participantSignUp.UserName,
			CreatedAt:  time.Now(),
		}
		if err := db.Postgresql.Create(&participantChannel).Error; err != nil {
			t.Logf("Warning: Failed to add participant to channel: %v", err)
		}

		joinURL := fmt.Sprintf("/api/v1/buzz/%s/join", buzzID)
		joinReq, err := http.NewRequest(http.MethodPost, joinURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		joinReq.Header.Set("Content-Type", "application/json")
		joinReq.Header.Set("Authorization", "Bearer "+participantToken)

		joinRR := httptest.NewRecorder()
		router.ServeHTTP(joinRR, joinReq)

		tst.AssertStatusCode(t, joinRR.Code, http.StatusOK)

		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("failed to fetch buzz: %v", err)
		}

		if buzz.OrgID == nil {
			t.Error("expected org_id to still be set after join")
		}
	})

	t.Run("EndOrgBuzzSuccessfully", func(t *testing.T) {
		channelID := utility.GenerateUUID()
		channel := models.Channels{
			ID:             channelID,
			Name:           "test_" + utility.GenerateUUID(),
			OrganisationID: hostUser.CurrentOrg.String(),
			OwnerId:        hostUser.ID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&channel).Error; err != nil {
			t.Fatalf("Failed to create test channel: %v", err)
		}

		userChannel := models.UserChannels{
			ChannelsID: channelID,
			UserID:     hostUser.ID,
			Username:   hostSignUp.UserName,
			CreatedAt:  time.Now(),
		}
		if err := db.Postgresql.Create(&userChannel).Error; err != nil {
			t.Logf("Warning: Failed to add user to channel: %v", err)
		}

		createBuzzData := models.CreateBuzzRequest{}

		jsonData, _ := json.Marshal(createBuzzData)
		createURL := "/api/v1/buzz/org/create"
		createReq, err := http.NewRequest(http.MethodPost, createURL, bytes.NewBuffer(jsonData))
		if err != nil {
			t.Fatal(err)
		}

		createReq.Header.Set("Content-Type", "application/json")
		createReq.Header.Set("Authorization", "Bearer "+hostToken)

		createRR := httptest.NewRecorder()
		router.ServeHTTP(createRR, createReq)

		if createRR.Code != http.StatusCreated {
			t.Fatalf("failed to create org buzz: %d", createRR.Code)
		}

		createData := tst.ParseResponse(createRR)
		createDataM := createData["data"].(map[string]any)
		buzzID := createDataM["buzz_id"].(string)

		endURL := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		endReq, err := http.NewRequest(http.MethodPost, endURL, nil)
		if err != nil {
			t.Fatal(err)
		}

		endReq.Header.Set("Content-Type", "application/json")
		endReq.Header.Set("Authorization", "Bearer "+hostToken)

		endRR := httptest.NewRecorder()
		router.ServeHTTP(endRR, endReq)

		tst.AssertStatusCode(t, endRR.Code, http.StatusOK)

		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("failed to fetch buzz: %v", err)
		}

		if buzz.Status != models.BuzzStatusEnded {
			t.Errorf("expected buzz status to be 'ended', got %s", buzz.Status)
		}

		if buzz.OrgID == nil {
			t.Error("expected org_id to still be set after ending")
		}
	})
}

func TestOrgBuzzMetadata(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	auth := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	// Host setup
	hostEmail := utility.GenerateUUID() + "@qa.team"
	hostSignUp := models.CreateUserRequestModel{
		Email:       hostEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "HostUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("hostuser_%v", utility.GenerateUUID())}
	hostLogin := models.LoginRequestModel{Email: hostSignUp.Email, Password: hostSignUp.Password}
	tst.SignupUser(t, gin.Default(), auth, hostSignUp, false)
	hostToken := tst.GetLoginToken(t, gin.Default(), auth, hostLogin)

	var hostUser models.User
	if err := db.Postgresql.Where("email = ?", hostSignUp.Email).First(&hostUser).Error; err != nil {
		t.Fatalf("failed to fetch host user: %v", err)
	}

	// Create a dummy channel to satisfy initial create requirements if any (though Org Buzz creates its own dummy)
	channelID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             channelID,
		Name:           "test_" + utility.GenerateUUID(),
		OrganisationID: hostUser.CurrentOrg.String(),
		OwnerId:        hostUser.ID,
		CreatedAt:      time.Now(),
	}
	db.Postgresql.Create(&channel)
	db.Postgresql.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     hostUser.ID,
		Username:   hostSignUp.UserName,
		CreatedAt:  time.Now(),
	})

	router, _ := SetupBuzzEndTestRouter(logger, validatorRef)

	// Create Org Buzz
	createBuzzData := models.CreateBuzzRequest{}
	jsonData, _ := json.Marshal(createBuzzData)
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/org/create", bytes.NewBuffer(jsonData))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+hostToken)
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("Failed to create Org Buzz: %d", createRR.Code)
	}

	createData := tst.ParseResponse(createRR)
	createDataM := createData["data"].(map[string]any)
	buzzID := createDataM["buzz_id"].(string)

	t.Run("GetMetadata_ShouldSucceedForOrgBuzz", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/buzz/%s/metadata", buzzID), nil)
		req.Header.Set("Authorization", "Bearer "+hostToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("GetBuzzMetadata failed for Org Buzz: expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})
}
