package test_shares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/shares"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateShares(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Setup user
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("shareuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "share",
		LastName:    "user",
		Password:    "password123",
		UserName:    fmt.Sprintf("shareuser%v", currUUID),
	}

	authController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	tst.SignupUser(t, r, authController, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: userSignUpData.Email, Password: userSignUpData.Password})

	tests := []struct {
		Name         string
		RequestBody  models.ShareRequest
		ExpectedCode int
		Message      string
	}{
		{
			Name:         "success - buy shares",
			RequestBody:  models.ShareRequest{NumberOfShares: 10, PricePurchased: 1.50},
			ExpectedCode: http.StatusCreated,
			Message:      "share created successfully",
		},
		{
			Name:         "edge - zero shares",
			RequestBody:  models.ShareRequest{NumberOfShares: 0, PricePurchased: 1.50},
			ExpectedCode: http.StatusUnprocessableEntity,
			Message:      "Validation failed",
		},
		{
			Name:         "edge - negative price",
			RequestBody:  models.ShareRequest{NumberOfShares: 5, PricePurchased: -1.00},
			ExpectedCode: http.StatusUnprocessableEntity,
			Message:      "Validation failed",
		},
		{
			Name:         "edge - missing fields",
			RequestBody:  models.ShareRequest{},
			ExpectedCode: http.StatusUnprocessableEntity,
			Message:      "Validation failed",
		},
	}

	sharesController := shares.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r := gin.Default()
			sharesURL := r.Group("/api/v1/shares", middleware.Authorize(db.Postgresql))
			sharesURL.POST("/", sharesController.Create)

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/shares/", &b)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)
			data := tst.ParseResponse(rr)
			if msg, ok := data["message"].(string); ok && test.Message != "" {
				tst.AssertResponseMessage(t, msg, test.Message)
			}
		})
	}
}

func TestDeleteShares(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// Setup user
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("delshareuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "del",
		LastName:    "user",
		Password:    "password123",
		UserName:    fmt.Sprintf("delshareuser%v", currUUID),
	}

	authController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	tst.SignupUser(t, r, authController, userSignUpData, false)
	token := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: userSignUpData.Email, Password: userSignUpData.Password})

	sharesController := shares.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	tests := []struct {
		Name         string
		ShareID      string
		ExpectedCode int
	}{
		{
			Name:         "edge - delete non-existent share",
			ShareID:      utility.GenerateUUID(),
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "edge - delete with invalid uuid",
			ShareID:      "invalid-uuid",
			ExpectedCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r := gin.Default()
			sharesURL := r.Group("/api/v1/shares", middleware.Authorize(db.Postgresql))
			sharesURL.DELETE("/:id", sharesController.Delete)

			deleteURI := url.URL{Path: fmt.Sprintf("/api/v1/shares/%s", test.ShareID)}
			req, _ := http.NewRequest(http.MethodDelete, deleteURI.String(), nil)
			req.Header.Set("Authorization", "Bearer "+token)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)
		})
	}
}

func TestGetSharesUnauthorized(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	sharesController := shares.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	sharesURL := r.Group("/api/v1/shares", middleware.Authorize(db.Postgresql))
	sharesURL.GET("/", sharesController.GetMyShares)

	// No auth token
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/shares/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
}
