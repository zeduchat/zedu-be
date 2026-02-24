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

func TestGetAUserForMentions(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user1 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "User One",
		Email:    fmt.Sprintf("user1_%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	user2 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "User Two",
		Email:    fmt.Sprintf("user2_%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	user3 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "User Three",
		Email:    fmt.Sprintf("user3_%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}

	org1 := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    "Org 1",
		Email:   fmt.Sprintf("org1_%v@qa.team", utility.GenerateUUID()),
		OwnerID: user1.ID,
	}

	db.Create(&user1)
	db.Create(&user2)
	db.Create(&user3)
	db.Create(&org1)

	// Add user1 and user2 to org1
	db.Create(&UserOrganisation{UserID: user1.ID, OrganisationID: org1.ID})
	db.Create(&UserOrganisation{UserID: user2.ID, OrganisationID: org1.ID})

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}
		return router, &authController
	}

	t.Run("Successful Get User For Mentions (Same Org)", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/mentions/%s", user2.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "User retrieved successfully")

		data := response["data"].(map[string]interface{})
		if data["userid"] != user2.ID {
			t.Errorf("expected userid %s, got %s", user2.ID, data["userid"])
		}
	})

	t.Run("Forbidden Get User For Mentions (Different Org)", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		// user3 is not in org1
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/mentions/%s", user3.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
	})

	t.Run("User Not Found", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, gin.Default(), *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/mentions/%s", utility.GenerateUUID()), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden) // Or 404 depending on implementation, but 403 because it checks org first
	})
}
