package test_integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/tests"
)

func TestIntegrationFlow(t *testing.T) {
	_, integrationController := SetupIntegrationTestRouter()
	db := integrationController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, err := utility.HashPassword("password")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	db.Create(&regularUser)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, _ := SetupIntegrationTestRouter()
		authController := auth.Controller{
			Db:        integrationController.Db,
			Validator: integrationController.Validator,
			Logger:    integrationController.Logger,
			ExtReq:    integrationController.ExtReq,
		}
		return router, &authController
	}

	t.Run("Successfully Create Integration App", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}

		token := tests.GetLoginToken(t, router, *authController, loginData)

		integrationApp := models.Integrations{
			Name:           fmt.Sprintf("IntegrationApp%v", currUUID),
			ApiEndpointUrl: "http://api.endpoint.com",
			AuthCredential: "some-auth-credential",
		}

		integrationAppJSON, _ := json.Marshal(integrationApp)

		req, err := http.NewRequest(http.MethodPost, "/api/v1/integration", bytes.NewBuffer(integrationAppJSON))
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
	})

	t.Run("Successfully Get All Integration Apps", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}

		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/integration", nil)
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})
}
