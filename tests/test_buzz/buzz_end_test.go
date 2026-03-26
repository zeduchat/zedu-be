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

func TestBuzzEnd(t *testing.T) {
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

	// Create host user
	hostEmail := utility.GenerateUUID() + "@qa.team"
	hostSignUp := models.CreateUserRequestModel{
		Email:       hostEmail,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "DMUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("dmuser1_%v", utility.GenerateUUID())}

	hostLogin := models.LoginRequestModel{Email: hostSignUp.Email, Password: hostSignUp.Password}

	r := gin.Default()
	tst.SignupUser(t, r, auth, hostSignUp, false)
	hostToken := tst.GetLoginToken(t, r, auth, hostLogin)
	if hostToken == "" {
		t.Fatalf("failed to obtain host login token")
	}

	router, _ := SetupBuzzEndTestRouter(logger, validatorRef)
	var hostUser models.User
	if err := db.Postgresql.Where("email = ?", hostSignUp.Email).First(&hostUser).Error; err != nil {
		t.Fatalf("failed to fetch host user: %v", err)
	}

	// Manually create channel and add user (bypass Elasticsearch issue)
	channelID := utility.GenerateUUID()

	// Create channel directly in database
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

	// Add user to channel
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

	t.Run("EndBuzzSuccessByNonHost", func(t *testing.T) {
		// Create a second user (non-host)
		nonHostEmail := utility.GenerateUUID() + "@qa.team"
		nonHostSignUp := models.CreateUserRequestModel{Email: nonHostEmail, Password: "password"}
		nonHostLogin := models.LoginRequestModel{Email: nonHostEmail, Password: "password"}

		tst.SignupUser(t, router, auth, nonHostSignUp, false)
		nonHostToken := tst.GetLoginToken(t, router, auth, nonHostLogin)
		if nonHostToken == "" {
			t.Fatalf("failed to obtain non-host login token")
		}

		// Create a new buzz for this test to avoid conflict
		newBuzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if newBuzzID == "" {
			t.Fatal("failed to obtain newBuzzID")
		}

		// Try to end buzz as non-host
		url := fmt.Sprintf("/api/v1/buzz/%s/end", newBuzzID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+nonHostToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		data := tst.ParseResponse(rr)
		dataM, ok := data["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data field in response, got %v", data)
		}

		if dataM["buzz_id"].(string) != newBuzzID {
			t.Errorf("expected buzz_id %s, got %s", newBuzzID, dataM["buzz_id"].(string))
		}
	})

	t.Run("EndBuzzSuccessByHost", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
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
		dataM, ok := data["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data field in response, got %v", data)
		}

		if dataM["buzz_id"].(string) != buzzID {
			t.Errorf("expected buzz_id %s, got %s", buzzID, dataM["buzz_id"].(string))
		}

		if dataM["status"].(string) != "ended" {
			t.Errorf("expected status 'ended', got %s", dataM["status"].(string))
		}

		if dataM["host_id"].(string) != hostUser.ID {
			t.Errorf("expected host_id %s, got %s", hostUser.ID, dataM["host_id"].(string))
		}

		// Verify buzz is actually ended in database
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
	t.Run("EndBuzzFailsWhenAlreadyEnded", func(t *testing.T) {
		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)

		// End buzz first time
		req1, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("Authorization", "Bearer "+hostToken)

		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req1)
		tst.AssertStatusCode(t, rr1.Code, http.StatusOK)

		// Try to end buzz second time
		req2, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+hostToken)

		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req2)

		tst.AssertStatusCode(t, rr2.Code, http.StatusConflict)

		data := tst.ParseResponse(rr2)
		message := data["message"].(string)
		if message != "buzz has ended" {
			t.Errorf("expected error message 'buzz has ended', got %s", message)
		}
	})

	t.Run("EndBuzzFailsWithInvalidBuzzID", func(t *testing.T) {
		invalidBuzzID := utility.GenerateUUID()
		url := fmt.Sprintf("/api/v1/buzz/%s/end", invalidBuzzID)
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
		if message != "buzz does not exist" {
			t.Errorf("expected error message 'buzz does not exist', got %s", message)
		}
	})

	t.Run("EndBuzzFailsWithMalformedBuzzID", func(t *testing.T) {
		url := "/api/v1/buzz/not-a-uuid/end"
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

	t.Run("EndBuzzFailsWithoutAuthentication", func(t *testing.T) {
		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, hostToken)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/%s/end", buzzID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})

	t.Run("EndBuzzUpdatesAllParticipantsStatus", func(t *testing.T) {
		// Verify all participants are marked as left
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
