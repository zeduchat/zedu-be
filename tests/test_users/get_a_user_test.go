package test_users

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

func TestGetAUser(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user1 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Vic Radon",
		Email:    fmt.Sprintf("vicradon%v@qa1.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}

	user2 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Target User",
		Email:    fmt.Sprintf("target%v@qa1.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}

	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    "Test Org",
		Email:   fmt.Sprintf("org%v@qa1.team", currUUID),
		OwnerID: user1.ID,
	}

	db.Create(&user1)
	db.Create(&user2)
	db.Create(&org)

	userOrg1 := UserOrganisation{
		UserID:         user1.ID,
		OrganisationID: org.ID,
	}
	userOrg2 := UserOrganisation{
		UserID:         user2.ID,
		OrganisationID: org.ID,
	}
	db.Create(&userOrg1)
	db.Create(&userOrg2)

	profModel := models.Profile{}
	_, _ = profModel.GetOrCreateProfileForOrg(db, user2.ID, org.ID)

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}
		return router, &authController
	}

	t.Run("Successful Get A User Profile", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s", user2.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "User retrieved successfully")

		data, ok := response["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data field in response")
		}

		if data["user_id"] != user2.ID {
			t.Errorf("expected user_id %s, got %v", user2.ID, data["user_id"])
		}
		if data["email"] != user2.Email {
			t.Errorf("expected email %s, got %v", user2.Email, data["email"])
		}
	})

	t.Run("Successful Get A User Profile via /user/:user_id", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/user/%s", user2.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "User retrieved successfully")
	})

	t.Run("User Not Found", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%s", utility.GenerateUUID()), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusNotFound)
	})
}
