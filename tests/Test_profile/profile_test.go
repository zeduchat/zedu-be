package test_profile

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestProfileFlow(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql
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
		router, profileController := SetupProfileTestRouter()
		authController := auth.Controller{
			Db:        profileController.Db,
			Validator: profileController.Validator,
			Logger:    profileController.Logger,
			ExtReq:    profileController.ExtReq,
		}
		return router, &authController
	}

	t.Run("Successfully Get User Profile", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
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