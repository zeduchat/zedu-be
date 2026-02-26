package test_users

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
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSwitchUserOrg_OrgRoleIDPersisted(t *testing.T) {
	router, userController := SetupUsersTestRouter()
	db := userController.Db.Postgresql

	// Register switch-org route
	router.PUT("/api/v1/users/switch-org",
		middleware.Authorize(db),
		middleware.CheckIsDeactivated(db),
		userController.SwitchUserOrg,
	)

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	// Create user
	testUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Switch Org Test User",
		Email:    fmt.Sprintf("switchorg%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}
	db.Create(&testUser)

	authCtrl := auth.Controller{
		Db:        userController.Db,
		Validator: userController.Validator,
		Logger:    userController.Logger,
		ExtReq:    userController.ExtReq,
	}
	orgCtrl := organisation.Controller{
		Db:        userController.Db,
		Validator: userController.Validator,
		Logger:    userController.Logger,
		ExtReq:    userController.ExtReq,
	}

	loginData := models.LoginRequestModel{
		Email:    testUser.Email,
		Password: "password",
	}
	token := tests.GetLoginToken(t, gin.Default(), authCtrl, loginData)

	// Create two organisations using separate routers to avoid route conflict
	org1Data := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("Org1-%s", currUUID),
		Email:       fmt.Sprintf("org1-%v@qa.team", currUUID),
		Description: "First org",
		Type:        "type1",
		Location:    "test",
		Country:     "test",
	}
	org2Data := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("Org2-%s", currUUID),
		Email:       fmt.Sprintf("org2-%v@qa.team", currUUID),
		Description: "Second org",
		Type:        "type1",
		Location:    "test",
		Country:     "test",
	}

	org1ID, _, _ := tests.CreateOrganisation(t, gin.Default(), userController.Db, orgCtrl, org1Data, token)
	org2ID, _, _ := tests.CreateOrganisation(t, gin.Default(), userController.Db, orgCtrl, org2Data, token)

	if org1ID == "" || org2ID == "" {
		t.Fatal("Failed to create test organisations")
	}

	// Get the role assigned to user in each org (via OrgUserManagement)
	var orgMgt1 models.OrgUserManagement
	if err := db.Where("user_id = ? AND organisation_id = ?", testUser.ID, org1ID).First(&orgMgt1).Error; err != nil {
		t.Fatalf("Failed to get OrgUserManagement for org1: %v", err)
	}

	var orgMgt2 models.OrgUserManagement
	if err := db.Where("user_id = ? AND organisation_id = ?", testUser.ID, org2ID).First(&orgMgt2).Error; err != nil {
		t.Fatalf("Failed to get OrgUserManagement for org2: %v", err)
	}

	t.Run("OrgRoleID is persisted to DB after switch", func(t *testing.T) {
		// Switch to org2
		switchReq := models.SwitchUserOrgReqeust{CurrentOrg: org2ID}
		body, _ := json.Marshal(switchReq)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/switch-org", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)

		// Verify the user's OrgRoleID in DB matches org2's role
		var dbUser models.User
		if err := db.Where("id = ?", testUser.ID).First(&dbUser).Error; err != nil {
			t.Fatalf("Failed to fetch user from DB: %v", err)
		}

		if dbUser.OrgRoleID == nil {
			t.Fatal("OrgRoleID is nil in DB after org switch — was not persisted")
		}

		if *dbUser.OrgRoleID != orgMgt2.RoleID {
			t.Errorf("OrgRoleID mismatch after switch to org2: got %q, want %q", *dbUser.OrgRoleID, orgMgt2.RoleID)
		}

		if dbUser.CurrentOrg.String() != org2ID {
			t.Errorf("CurrentOrg mismatch: got %q, want %q", dbUser.CurrentOrg.String(), org2ID)
		}
	})

	t.Run("OrgRoleID updates correctly on second switch", func(t *testing.T) {
		// Extract the new token from the previous switch (old token was revoked)
		// Re-login to get a fresh token since previous switch revoked the old one
		freshToken := tests.GetLoginToken(t, gin.Default(), authCtrl, loginData)

		// Switch to org1
		switchReq := models.SwitchUserOrgReqeust{CurrentOrg: org1ID}
		body, _ := json.Marshal(switchReq)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/users/switch-org", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", freshToken))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)

		// Verify the user's OrgRoleID in DB now matches org1's role
		var dbUser models.User
		if err := db.Where("id = ?", testUser.ID).First(&dbUser).Error; err != nil {
			t.Fatalf("Failed to fetch user from DB: %v", err)
		}

		if dbUser.OrgRoleID == nil {
			t.Fatal("OrgRoleID is nil after second org switch")
		}

		if *dbUser.OrgRoleID != orgMgt1.RoleID {
			t.Errorf("OrgRoleID mismatch after switch to org1: got %q, want %q", *dbUser.OrgRoleID, orgMgt1.RoleID)
		}

		if dbUser.CurrentOrg.String() != org1ID {
			t.Errorf("CurrentOrg mismatch: got %q, want %q", dbUser.CurrentOrg.String(), org1ID)
		}
	})
}
