package test_buzz

import (
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
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestEndBuzzByChannel(t *testing.T) {
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

	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}

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

	t.Run("EndBuzzByChannelSuccessByHost", func(t *testing.T) {
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

		createBuzzData := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/channel/%s/end", channelID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)

		if dataM["buzz_id"].(string) != buzzID {
			t.Errorf("expected buzz_id %s, got %s", buzzID, dataM["buzz_id"].(string))
		}

		if dataM["status"].(string) != "ended" {
			t.Errorf("expected status 'ended', got %s", dataM["status"].(string))
		}

		if dataM["channel_id"].(string) != channelID {
			t.Errorf("expected channel_id %s, got %s", channelID, dataM["channel_id"].(string))
		}

		var buzz models.Buzz
		if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
			t.Fatalf("failed to fetch buzz from database: %v", err)
		}

		if buzz.Status != models.BuzzStatusEnded {
			t.Errorf("expected buzz status to be 'ended', got %s", buzz.Status)
		}

		if buzz.IsLiveStatus {
			t.Error("expected is_live_status to be false")
		}

		if buzz.BuzzEndTime == nil {
			t.Error("expected buzz_end_time to be set")
		}
	})

	t.Run("EndBuzzByChannelFailsWhenNonHostAttempts", func(t *testing.T) {
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

		createBuzzData := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		nonHostEmail := utility.GenerateUUID() + "@qa.team"
		nonHostSignUp := models.CreateUserRequestModel{
			Email:    nonHostEmail,
			Password: "password",
			UserName: fmt.Sprintf("nonhost_%v", utility.GenerateUUID()),
		}
		nonHostLogin := models.LoginRequestModel{Email: nonHostSignUp.Email, Password: nonHostSignUp.Password}

		tst.SignupUser(t, gin.Default(), auth, nonHostSignUp, false)
		nonHostToken := tst.GetLoginToken(t, gin.Default(), auth, nonHostLogin)
		if nonHostToken == "" {
			t.Fatalf("failed to obtain non-host login token")
		}

		url := fmt.Sprintf("/api/v1/buzz/channel/%s/end", channelID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+nonHostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "only the buzz host can perform this action" {
			t.Errorf("expected error message 'only the buzz host can perform this action', got %s", message)
		}
	})

	t.Run("EndBuzzByChannelFailsWhenNoActiveBuzz", func(t *testing.T) {
		emptyChannelID := utility.GenerateUUID()

		emptyChannel := models.Channels{
			ID:             emptyChannelID,
			Name:           "empty_" + utility.GenerateUUID(),
			OrganisationID: hostUser.CurrentOrg.String(),
			OwnerId:        hostUser.ID,
			CreatedAt:      time.Now(),
		}
		if err := db.Postgresql.Create(&emptyChannel).Error; err != nil {
			t.Fatalf("Failed to create empty channel: %v", err)
		}

		url := fmt.Sprintf("/api/v1/buzz/channel/%s/end", emptyChannelID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)

		data := tst.ParseResponse(rr)
		message := data["message"].(string)
		if message != "no active buzz in this channel" {
			t.Errorf("expected error message 'no active buzz in this channel', got %s", message)
		}
	})

	t.Run("EndBuzzByChannelFailsWithInvalidChannelID", func(t *testing.T) {
		url := "/api/v1/buzz/channel/not-a-uuid/end"
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})

	t.Run("EndBuzzByChannelFailsWithoutAuthentication", func(t *testing.T) {
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

		createBuzzData := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/channel/%s/end", channelID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})

	t.Run("EndBuzzByChannelUpdatesAllParticipantsStatus", func(t *testing.T) {
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

		createBuzzData := models.CreateBuzzRequest{
			ChannelID: channelID,
		}

		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/channel/%s/end", channelID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+hostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		var participants []models.BuzzParticipant
		if err := db.Postgresql.Where("buzz_id = ?", buzzID).Find(&participants).Error; err != nil {
			t.Fatalf("failed to fetch participants: %v", err)
		}

		for _, p := range participants {
			if p.Status != models.BuzzParticipantStatusLeft {
				t.Errorf("expected participant %s status to be 'left', got %s", p.UserID, p.Status)
			}
		}
	})
}
