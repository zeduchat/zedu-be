package test_buzz

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateCamera(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authController := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	router, _ := SetupBuzzTestRouter(logger, validatorRef)

	userEmail := utility.GenerateUUID() + "@qa.team"
	signUp := models.CreateUserRequestModel{Email: userEmail, Password: "password"}
	login := models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password}

	tst.SignupUser(t, router, authController, signUp, false)
	token := tst.GetLoginToken(t, router, authController, login)
	if token == "" {
		t.Fatalf("failed to obtain login token")
	}

	var user models.User
	if err := db.Postgresql.Where("email = ?", signUp.Email).First(&user).Error; err != nil {
		t.Fatalf("failed to fetch created user: %v", err)
	}

	buzzID := utility.GenerateUUID()
	h := models.Buzz{ID: buzzID, ChannelID: utility.GenerateUUID(), HostID: user.ID, ParticipantIDs: pq.StringArray{user.ID}, BuzzStartTime: time.Now().UTC()}
	if err := db.Postgresql.Create(&h).Error; err != nil {
		t.Fatalf("failed to create buzz: %v", err)
	}

	hp := models.BuzzParticipant{BuzzID: buzzID, UserID: user.ID}
	if err := db.Postgresql.Create(&hp).Error; err != nil {
		t.Fatalf("failed to create buzz participant: %v", err)
	}

	t.Run("UpdateCameraSuccess", func(t *testing.T) {
		reqBody := models.UpdateCameraRequest{UserID: user.ID, Status: true}
		b, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/buzz/"+buzzID+"/camera", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
	})

	t.Run("ForbiddenWhenTogglingOtherUser", func(t *testing.T) {
		otherEmail := utility.GenerateUUID() + "@qa.team"
		otherSign := models.CreateUserRequestModel{Email: otherEmail, Password: "password"}

		tst.SignupUser(t, gin.Default(), authController, otherSign, false)
		var other models.User
		if err := db.Postgresql.Where("email = ?", otherSign.Email).First(&other).Error; err != nil {
			t.Fatalf("failed to fetch other user: %v", err)
		}

		reqBody := models.UpdateCameraRequest{UserID: other.ID, Status: true}
		b, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/buzz/"+buzzID+"/camera", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("NonParticipantReturns404", func(t *testing.T) {
		nonEmail := utility.GenerateUUID() + "@qa.team"
		nonSign := models.CreateUserRequestModel{Email: nonEmail, Password: "password"}
		nonLogin := models.LoginRequestModel{Email: nonSign.Email, Password: nonSign.Password}
		tst.SignupUser(t, gin.Default(), authController, nonSign, false)
		nonToken := tst.GetLoginToken(t, gin.Default(), authController, nonLogin)
		if nonToken == "" {
			t.Fatalf("failed to get token for non-participant")
		}

		var non models.User
		if err := db.Postgresql.Where("email = ?", nonSign.Email).First(&non).Error; err != nil {
			t.Fatalf("failed to fetch non participant user: %v", err)
		}

		reqBody := models.UpdateCameraRequest{UserID: non.ID, Status: true}
		b, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/buzz/"+buzzID+"/camera", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+nonToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("InvalidBuzzIDReturns400", func(t *testing.T) {
		reqBody := models.UpdateCameraRequest{UserID: user.ID, Status: true}
		b, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/buzz/invalid-id/camera", bytes.NewBuffer(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})
}
