package test_users

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
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

	setup := func() (*gin.Engine, *auth.Controller) {
		authController := auth.Controller{
			Db:        userController.Db,
			Validator: userController.Validator,
			Logger:    userController.Logger,
			ExtReq:    userController.ExtReq,
		}
		return router, &authController
	}

	router, authController := setup()

	// 1. Signup user1 via endpoint to get default org
	user1Signup := models.CreateUserRequestModel{
		Email:     fmt.Sprintf("user1_%v@qa.team", currUUID),
		Password:  "password",
		FirstName: "User",
		LastName:  "One",
		UserName:  fmt.Sprintf("user1_%v", currUUID),
	}
	tests.SignupUser(t, router, *authController, user1Signup, false)

	// 2. Fetch user1 from DB to get their default OrgID
	var user1 models.User
	db.Where("email = ?", user1Signup.Email).First(&user1)

	if user1.CurrentOrg == uuid.Nil {
		var userOrg UserOrganisation
		db.Where("user_id = ?", user1.ID).First(&userOrg)
		if userOrg.OrganisationID != "" {
			user1.CurrentOrg, _ = uuid.FromString(userOrg.OrganisationID)
			db.Save(&user1)
		} else {
			t.Fatalf("failed to find default organization for user1")
		}
	}
	orgID := user1.CurrentOrg.String()

	// Fetch a valid role to avoid UUID syntax error in PostgreSQL
	var role models.OrgRole
	db.Where("name = ?", "User").First(&role)
	if role.ID == "" {
		db.First(&role)
	}

	// 3. Create user2 and user3 manually but link user2 to user1's org
	user2 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "User Two",
		Email:    fmt.Sprintf("user2_%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
		Profile: models.Profile{
			ID:       utility.GenerateUUID(),
			UserName: "user2_mention",
		},
	}
	user3 := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "User Three",
		Email:    fmt.Sprintf("user3_%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
		Profile: models.Profile{
			ID:       utility.GenerateUUID(),
			UserName: "user3_mention",
		},
	}
	db.Create(&user2)
	db.Create(&user3)

	// Add user2 to user1's org
	db.Create(&UserOrganisation{UserID: user2.ID, OrganisationID: orgID})
	db.Create(&models.OrgUserManagement{
		UserID:         user2.ID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         role.ID,
	})

	t.Run("Successful Get User For Mentions (Same Org)", func(t *testing.T) {
		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/mentions/%s", user2.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "User retrieved successfully")

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected data field in response, got nil or wrong type")
		}
		if data["userid"] != user2.ID {
			t.Errorf("expected userid %s, got %s", user2.ID, data["userid"])
		}
	})

	t.Run("Forbidden Get User For Mentions (Different Org)", func(t *testing.T) {
		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		// user3 is not in orgID
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/mentions/%s", user3.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
	})

	t.Run("User Not Found", func(t *testing.T) {
		loginData := models.LoginRequestModel{
			Email:    user1.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/mentions/%s", utility.GenerateUUID()), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
	})
}
