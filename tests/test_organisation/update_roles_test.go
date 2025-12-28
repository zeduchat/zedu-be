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

func TestUpdateOrgRole(t *testing.T) {
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

	roleID := utility.GenerateUUID()

	role := models.OrgRole{
		ID:             roleID,
		Name:           fmt.Sprintf("Admin Role-%v", utility.RandomString(5)),
		Description:    "Administrator role",
		OrganisationID: &orgID,
		IsDefault:      false,
	}

	db.Create(&role)

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

	t.Run("Successful Update Org Role", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		updatedRole := models.OrgRole{
			Name:        fmt.Sprintf("%v-New name", utility.RandomString(5)),
			Description: "Newdescription",
		}
		roleJSON, _ := json.Marshal(updatedRole)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s", orgID, roleID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Org role updated successfully")
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, _ := setup()

		updatedRole := models.OrgRole{
			Name:        fmt.Sprintf("%v-Another name", utility.RandomString(6)),
			Description: "Another role description",
		}
		roleJSON, _ := json.Marshal(updatedRole)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s", orgID, roleID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

	t.Run("Forbidden Access - Regular User Trying to Update Org Role", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		updatedRole := models.OrgRole{
			Name:        fmt.Sprintf("%v-Regur name", utility.RandomString(5)),
			Description: "Regular user role description",
		}
		roleJSON, _ := json.Marshal(updatedRole)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s", orgID, roleID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "requester is not the owner of the organisation")
	})

	t.Run("Bad Request - Validation Errors", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		invalidRole := models.OrgRole{
			Description: "Missing name field",
		}
		roleJSON, _ := json.Marshal(invalidRole)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s", orgID, roleID), bytes.NewBuffer(roleJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnprocessableEntity)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Validation failed")
	})
}

func TestUpdateOrgPermissions(t *testing.T) {
	_, orgController := SetupOrgTestRouter()
	db := orgController.Db.Postgresql
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

	roleID := utility.GenerateUUID()

	role := models.OrgRole{
		ID:             roleID,
		Name:           fmt.Sprintf("Admin Role-%v", utility.RandomString(5)),
		Description:    "Administrator role",
		OrganisationID: &orgID,
		IsDefault:      false,
	}

	perm := models.Permission{
		ID:        utility.GenerateUUID(),
		RoleID:    role.ID,
		IsDefault: false,
		PermissionList: models.PermissionList{
			CanRemovePeopleFromOrganization: false,
			CanInviteMembers:                true,
			CanCreateCustomRole:             false,
			CanCreateChannel:                true,
			CanCommentOnThreads:             true,
			CanViewBilling:                  false,
			CanCreateWebhooks:               false,
			CanViewChannels:                 true,
		},
	}

	db.Create(&role)
	db.Create(&perm)

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

	t.Run("Successful Update Org Permissions", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		updatedPermissions := models.PermissionList{
			CanRemovePeopleFromOrganization: false,
			CanInviteMembers:                true,
			CanCreateCustomRole:             false,
		}

		permissionsJSON, _ := json.Marshal(updatedPermissions)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s/permissions", orgID, roleID), bytes.NewBuffer(permissionsJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Permissions updated successfully")
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		router, _ := setup()

		updatedPermissions := models.Permission{
			PermissionList: models.PermissionList{
				CanRemovePeopleFromOrganization: true,
				CanInviteMembers:                true,
				CanCreateCustomRole:             false,
			},
		}
		permissionsJSON, _ := json.Marshal(updatedPermissions)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s/permissions", orgID, roleID), bytes.NewBuffer(permissionsJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusUnauthorized)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Token is invalid!")
	})

	t.Run("Forbidden Access - Regular User Trying to Update Org Permissions", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		updatedPermissions := models.Permission{
			PermissionList: models.PermissionList{
				CanRemovePeopleFromOrganization: false,
				CanInviteMembers:                true,
				CanCreateCustomRole:             false,
			},
		}
		permissionsJSON, _ := json.Marshal(updatedPermissions)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s/permissions", orgID, roleID), bytes.NewBuffer(permissionsJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusForbidden)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "not organization owner")
	})

	t.Run("Bad Request - bad body", func(t *testing.T) {
		router, orgController := setup()

		loginData := models.LoginRequestModel{
			Email:    adminUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *orgController, loginData)

		invalidPermissions := map[string]any{
			"permission_list": "invalid_permissions",
		}
		permissionsJSON, _ := json.Marshal(invalidPermissions)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/organisations/%s/roles/%s/permissions", orgID, roleID), bytes.NewBuffer(permissionsJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusBadRequest)
		response := tests.ParseResponse(resp)
		tests.AssertResponseMessage(t, response["message"].(string), "Failed to parse request body")
	})
}
