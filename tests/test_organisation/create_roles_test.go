package test_organisation

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
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateOrgRole(t *testing.T) {
	_, orgController := SetupOrgTestRouter()
	db := orgController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	adminUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Admin User",
		Email:    fmt.Sprintf("admin%v@qa.team", currUUID),
		Password: password,
	}

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	orgID := utility.GenerateUUID()

	org := models.Organisation{
		ID:      orgID,
		Name:    fmt.Sprintf("Org comp%v", currUUID),
		Email:   fmt.Sprintf("orgtest%v@qa.team", currUUID),
		OwnerID: adminUser.ID,
	}

	db.Create(&adminUser)
	db.Create(&regularUser)
	db.Create(&org)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, orgController := SetupOrgTestRouter()
		authController := auth.Controller{
			Db:        orgController.Db,
			Validator: orgController.Validator,
			Logger:    orgController.Logger,
			ExtReq:    orgController.ExtReq,
		}

		return router, &authController
	}

	t.Run("Successful Create Org Role", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		role := models.OrgRole{
			Name:        fmt.Sprintf("Admin Role-%v", utility.RandomString(5)),
			Description: "New role description",
		}
		roleJSON, _ := json.Marshal(role)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/roles", orgID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Org role created successfully")
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, _ := setup()

		role := models.OrgRole{
			Name:        fmt.Sprintf("Admin-%v", utility.RandomString(7)),
			Description: "New role description",
		}
		roleJSON, _ := json.Marshal(role)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/roles", orgID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

	t.Run("Forbidden Access - Regular User Trying to Create Org Role", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		role := models.OrgRole{
			Name:           fmt.Sprintf("Admin Role-%v", utility.RandomString(5)),
			Description:    "New role description",
			OrganisationID: &orgID,
		}
		roleJSON, _ := json.Marshal(role)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/roles", orgID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "you do not have permission to create roles")
	})

	t.Run("Successful Create Org Role - Non-Owner Administrator", func(t *testing.T) {
		router, orgController := setup()

		nonOwnerAdmin := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Non-Owner Admin",
			Email:    fmt.Sprintf("nonowneradmin%v@qa.team", utility.GenerateUUID()),
			Password: password,
		}
		db.Create(&nonOwnerAdmin)

		var adminRole models.OrgRole
		if err := db.Where("name = ?", models.OrgRoleNameAdministrator).First(&adminRole).Error; err != nil {
			t.Fatalf("Administrator role not found: %v", err)
		}

		orgUserMgt := models.OrgUserManagement{
			UserID:         nonOwnerAdmin.ID,
			OrganisationID: orgID,
			RoleID:         adminRole.ID,
		}
		db.Create(&orgUserMgt)

		loginData := models.LoginRequestModel{
			Email:    nonOwnerAdmin.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		role := models.OrgRole{
			Name:        fmt.Sprintf("Admin-%v", utility.RandomString(5)),
			Description: "Created by non-owner Administrator",
		}
		roleJSON, _ := json.Marshal(role)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/roles", orgID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusCreated)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Org role created successfully")
	})

	t.Run("Bad Request - Missing Fields", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		role := models.OrgRole{
			Description:    "Missing Name",
			OrganisationID: &orgID,
		}
		roleJSON, _ := json.Marshal(role)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/roles", orgID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})
}
