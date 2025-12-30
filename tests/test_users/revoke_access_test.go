package test_users

import (
	"bytes"
	"encoding/json"
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

func TestRevokeUserAccessToken(t *testing.T) {
	router, userController := SetupUsersTestRouter()
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

	accessToken := models.AccessToken{
		ID:                        utility.GenerateUUID(),
		OwnerID:                   adminUser.ID,
		IsLive:                    true,
		LoginAccessToken:          utility.RandomString(30),
		LoginAccessTokenExpiresIn: time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	db.Create(&accessToken)

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}

		return router, &authController
	}

	t.Run("Successful Revoke User Access Token", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		reqBody := models.TerminateSessionRequest{
			UserID:            &adminUser.ID,
			GlobalTermination: false,
			AccessToken:       &accessToken.ID,
		}
		reqBodyJSON, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/revoke-session", bytes.NewBuffer(reqBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "User session terminated successfully")
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, _ := setup()

		reqBody := models.TerminateSessionRequest{
			UserID:            &adminUser.ID,
			GlobalTermination: false,
			AccessToken:       &accessToken.ID,
		}
		reqBodyJSON, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/revoke-session", bytes.NewBuffer(reqBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

	t.Run("Forbidden Access - Regular User Trying to Revoke Session", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		reqBody := models.TerminateSessionRequest{
			UserID:            &adminUser.ID,
			GlobalTermination: false,
			AccessToken:       &accessToken.ID,
		}
		reqBodyJSON, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/revoke-session", bytes.NewBuffer(reqBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "user does not have permission to update this user")
	})

	t.Run("Bad Request - Bad Body", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		invalidReqBody := map[string]any{
			"user_id":            "invalid_user_id",
			"global_termination": "invalid_value",
			"access_token_id":    "invalid_access_token_id",
		}
		reqBodyJSON, _ := json.Marshal(invalidReqBody)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/revoke-session", bytes.NewBuffer(reqBodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Failed to parse request body")
	})
}
