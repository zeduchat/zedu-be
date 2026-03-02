package test_buzz

import (
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

func TestMuteParticipants(t *testing.T) {
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
	h := models.Buzz{
		ID:             buzzID,
		ChannelID:      utility.GenerateUUID(),
		OriginalHostID: user.ID,
		HostID:         user.ID,
		ParticipantIDs: pq.StringArray{user.ID},
		BuzzStartTime:  time.Now().UTC(),
		Status:         models.BuzzStatusActive,
		IsLiveStatus:   true,
	}
	if err := db.Postgresql.Create(&h).Error; err != nil {
		t.Fatalf("failed to create buzz: %v", err)
	}

	hp := models.BuzzParticipant{
		ID:      utility.GenerateUUID(),
		BuzzID:  buzzID,
		UserID:  user.ID,
		Status:  models.BuzzParticipantStatusActive,
		IsMuted: false,
	}
	if err := db.Postgresql.Create(&hp).Error; err != nil {
		t.Fatalf("failed to create buzz participant: %v", err)
	}

	t.Run("MuteParticipantsSuccess", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/mute-participants", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		var updatedParticipant models.BuzzParticipant
		db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, user.ID).First(&updatedParticipant)
		if !updatedParticipant.IsMuted {
			t.Errorf("expected participant to be muted, but is_muted is false")
		}
	})


	t.Run("MuteParticipantsForbiddenForNonHost", func(t *testing.T) {
		otherEmail := utility.GenerateUUID() + "@qa.team"
		otherSign := models.CreateUserRequestModel{Email: otherEmail, Password: "password"}
		otherLogin := models.LoginRequestModel{Email: otherSign.Email, Password: otherSign.Password}
		tst.SignupUser(t, router, authController, otherSign, false)
		otherToken := tst.GetLoginToken(t, router, authController, otherLogin)

		var otherUser models.User
		db.Postgresql.Where("email = ?", otherEmail).First(&otherUser)

		// Add other user as participant but NOT host
		hpOther := models.BuzzParticipant{
			ID:      utility.GenerateUUID(),
			BuzzID:  buzzID,
			UserID:  otherUser.ID,
			Status:  models.BuzzParticipantStatusActive,
			IsMuted: false,
		}
		db.Postgresql.Create(&hpOther)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/"+buzzID+"/mute-participants", nil)
		req.Header.Set("Authorization", "Bearer "+otherToken)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("MuteParticipantsInvalidBuzz", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/buzz/invalid-id/mute-participants", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
	})
}
