package test_users

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateUserRole(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	adminData := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "admin jane doe",
		Email:    fmt.Sprintf("testadmin%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}
	db.Create(&adminData)

	userData := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "user jane doe",
		Email:    fmt.Sprintf("testuser%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.User),
	}
	db.Create(&userData)

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:      orgID,
		Name:    fmt.Sprintf("Org comp%v", currUUID),
		Email:   fmt.Sprintf("orgtest%v@qa.team", currUUID),
		OwnerID: adminData.ID,
	}
	db.Create(&org)

	superAdminRole := models.OrgRole{
		ID:             utility.GenerateUUID(),
		Name:           fmt.Sprintf("Admin-%v", utility.RandomString(7)),
		Description:    "Super Administrator Role",
		OrganisationID: orgID,
	}

	userRole := models.OrgRole{
		ID:             utility.GenerateUUID(),
		Name:           fmt.Sprintf("Vendor-%v", utility.RandomString(5)),
		Description:    "Standard User Role",
		OrganisationID: orgID,
	}
	db.Create(&superAdminRole)
	db.Create(&userRole)

	loginData := models.LoginRequestModel{
		Email:    adminData.Email,
		Password: "password",
	}

	auth := auth.Controller{Db: userController.Db, Validator: userController.Validator, Logger: userController.Logger}
	token := tests.GetLoginToken(t, router, auth, loginData)

	t.Run("Successful Update", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/users/%s/roles/%s", userData.ID, superAdminRole.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Role updated successfully")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/users/%s/roles/%s", userData.ID, superAdminRole.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token could not be found!")
		tests.AssertResponseMessage(t, response["error"].(string), "Unauthorized")
	})

	t.Run("User Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/users/%s/roles/%s", currUUID, superAdminRole.ID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusNotFound)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "invalid user")
	})

	t.Run("Role Not Found", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/users/%s/roles/%s", userData.ID, currUUID), nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusNotFound)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "invalid role")
	})

}
