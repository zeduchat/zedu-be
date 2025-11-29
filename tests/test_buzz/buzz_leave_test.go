package test_buzz

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzzLeave(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authController := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelController := channel.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	buzzController := buzz.Controller{Db: db, Validator: validatorRef, Logger: logger}
	router, _ := SetupBuzzTestRouter()

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
	channelData := models.CreateChannelsRequest{
		OrganisationID: user.CurrentOrg.String(),
		Username:       user.Profile.UserName,
		Name:           "test_" + utility.GenerateUUID(),
	}
	channelID, _ := tst.CreateChannels(t, router, channelController, db, channelData, token)
	if channelID == "" {
		t.Fatal("failed to obtain channelID")
	}

	createBuzzData := models.CreateBuzzRequest{
		ChannelID: channelID,
	}

	t.Run("LeaveBuzzSuccess", func(t *testing.T) {
		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, token)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/%s/leave", buzzID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	})

	/**
		Currently only host is in the buzz,
		if host leaves then should should end
	**/
	t.Run("MeetingEndedSuccess", func(t *testing.T) {
		buzzID, _ := tst.CreateBuzz(t, router, buzzController, db, createBuzzData, token)
		if buzzID == "" {
			t.Fatal("failed to obtain buzzID")
		}

		url := fmt.Sprintf("/api/v1/buzz/%s/leave", buzzID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		data := tst.ParseResponse(rr)
		dataM := data["data"].(map[string]any)
		buzzEnded := dataM["buzz_ended"].(bool)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		if !(buzzEnded) {
			t.Errorf("expected the call to have ended")
		}

	})

	t.Run("InvalidBuzzIDReturns400", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/buzz/%s/leave", utility.GenerateUUID())
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			t.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)

	})

}
