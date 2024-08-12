package test_users

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserLoginAudit(t *testing.T) {
	_, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	adminUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Admin User",
		Email:    fmt.Sprintf("admin%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}

	db.Create(&adminUser)
	db.Create(&regularUser)

	loginAudits := []models.LoginActivity{
		{
			ID:        utility.GenerateUUID(),
			UserID:    adminUser.ID,
			LoginAt:   time.Now(),
			IPAddress: "192.168.1.1",
			Location:  "Location 1",
			Device:    "Browser 1",
		},
		{
			ID:        utility.GenerateUUID(),
			UserID:    adminUser.ID,
			LoginAt:   time.Now(),
			IPAddress: "192.168.1.1",
			Location:  "Location 2",
			Device:    "Browser 2",
		},
	}

	for _, audit := range loginAudits {
		db.Create(&audit)
	}

	setup := func() (*gin.Engine, *auth.Controller) {
		router, userController := SetupUsersTestRouter()
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("Successful Get User Login Audit", func(t *testing.T) {
		router, userController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *userController, loginData)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/login-audit", adminUser.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Data retrieved successfully")
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, _ := setup()

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/login-audit", adminUser.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

	t.Run("Forbidden Access - Regular User Trying to Get Another User's Login Audit", func(t *testing.T) {
		router, userController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *userController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s/login-audit", adminUser.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "login activities not found")
	})
}
