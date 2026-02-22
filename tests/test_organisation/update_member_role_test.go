package test_organisation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// setupOrgWithMembers creates:
//   - An owner who creates an org (gets Administrator role)
//   - A second user added to the org with the "User" role
//
// Returns orgID, ownerToken, memberUserID, memberToken.
func setupOrgWithMembers(t *testing.T) (string, string, string, string) {
	t.Helper()

	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	ensureCanChangeUserOrgRoleSeeded(t, db)

	// --- Owner ---
	ownerUUID := utility.GenerateUUID()
	ownerSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("owner_%v@qa.team", ownerUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Owner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("owner_%v", ownerUUID),
	}
	ownerLogin := models.LoginRequestModel{
		Email:    ownerSignUp.Email,
		Password: ownerSignUp.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, ownerSignUp, false)
	ownerToken := tst.GetLoginToken(t, r, authCtrl, ownerLogin)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("RoleTestOrg%s", ownerUUID),
		Description: "Org for testing role updates",
		Email:       ownerSignUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, ownerToken)

	// Re-login to get fresh token with correct org_id
	r2 := gin.Default()
	ownerToken = tst.GetLoginToken(t, r2, authCtrl, ownerLogin)

	// --- Member ---
	memberUUID := utility.GenerateUUID()
	memberSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("member_%v@qa.team", memberUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Member",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("member_%v", memberUUID),
	}
	memberLogin := models.LoginRequestModel{
		Email:    memberSignUp.Email,
		Password: memberSignUp.Password,
	}

	r3 := gin.Default()
	tst.SignupUser(t, r3, authCtrl, memberSignUp, false)
	memberToken := tst.GetLoginToken(t, r3, authCtrl, memberLogin)
	memberUserID := tst.GetUserIDFromToken(t, memberToken, db)

	// Add member to org with "User" role
	userRoleID := getRoleIDByName(t, db, "User")
	addMemberViaEndpoint(t, db, orgCtrl, orgID, memberUserID, userRoleID, ownerToken)

	// Switch member's current org and re-login
	db.Postgresql.Model(&models.User{}).Where("id = ?", memberUserID).
		Update("current_org", orgID)
	r4 := gin.Default()
	memberToken = tst.GetLoginToken(t, r4, authCtrl, memberLogin)

	return orgID, ownerToken, memberUserID, memberToken
}

// ensureCanChangeUserOrgRoleSeeded ensures the Administrator role has can_change_user_org_role set.
func ensureCanChangeUserOrgRoleSeeded(t *testing.T, db *storage.Database) {
	t.Helper()
	var adminRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", "Administrator").First(&adminRole).Error; err != nil {
		t.Fatalf("failed to find Administrator role: %v", err)
	}
	var perm models.Permission
	if err := db.Postgresql.Where("role_id = ?", adminRole.ID).First(&perm).Error; err != nil {
		t.Fatalf("failed to find permission for Administrator role: %v", err)
	}
	if !perm.PermissionList.CanChangeUserOrgRole {
		perm.PermissionList.CanChangeUserOrgRole = true
		db.Postgresql.Save(&perm)
	}
	// Invalidate Redis cache
	rd.RedisDelete(db.Redis, "role_permissions_"+adminRole.ID)
}

func getRoleIDByName(t *testing.T, db *storage.Database, name string) string {
	t.Helper()
	var role models.OrgRole
	err := db.Postgresql.Where("name = ?", name).First(&role).Error
	if err != nil {
		t.Fatalf("failed to get %s role: %v", name, err)
	}
	return role.ID
}

func addMemberViaEndpoint(t *testing.T, db *storage.Database, orgCtrl organisation.Controller, orgID, memberUserID, roleID, adminToken string) {
	t.Helper()
	r := gin.Default()
	orgUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	orgUrl.POST("/organisations/:org_id/members", orgCtrl.AddMemberToOrganisation)

	body := models.OrgUserCreateRequest{
		UserID: memberUserID,
		RoleID: roleID,
	}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/members", orgID), &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Logf("Warning: Add member returned status %d: %s", rr.Code, rr.Body.String())
	}
}

// callUpdateMemberRole calls PATCH /organisations/:org_id/users/:user_id/role
// with PermissionMiddleware enforcing can_change_user_org_role.
func callUpdateMemberRole(t *testing.T, db *storage.Database, orgCtrl organisation.Controller, orgID, targetUserID, newRoleID, token string) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.Default()
	orgUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	orgUrl.PATCH("/organisations/:org_id/users/:user_id/role",
		middleware.PermissionMiddleware(db.Postgresql, db.Redis, "can_change_user_org_role"),
		orgCtrl.UpdateMemberRole,
	)

	body := models.UpdateMemberRoleRequest{RoleID: newRoleID}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, err := http.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/v1/organisations/%s/users/%s/role", orgID, targetUserID), &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// getMemberRoleID returns the current role_id for a user in an org.
func getMemberRoleID(t *testing.T, db *storage.Database, orgID, userID string) string {
	t.Helper()
	var oum models.OrgUserManagement
	err := db.Postgresql.Where("organisation_id = ? AND user_id = ?", orgID, userID).First(&oum).Error
	if err != nil {
		t.Fatalf("failed to get member role: %v", err)
	}
	return oum.RoleID
}

// TestUpdateMemberRole_AdminCanChangeRole verifies that an org admin (owner)
// can change a member's role via the PermissionMiddleware-protected endpoint.
func TestUpdateMemberRole_AdminCanChangeRole(t *testing.T) {
	orgID, ownerToken, memberUserID, _ := setupOrgWithMembers(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	userRoleID := getRoleIDByName(t, db, "User")
	managerRoleID := getRoleIDByName(t, db, "Manager")

	// Verify the member currently has "User" role
	t.Run("Member starts with User role", func(t *testing.T) {
		currentRole := getMemberRoleID(t, db, orgID, memberUserID)
		if currentRole != userRoleID {
			t.Errorf("expected member to start with User role %s, got %s", userRoleID, currentRole)
		}
	})

	// Owner changes member's role to Manager
	t.Run("Owner can change member role to Manager", func(t *testing.T) {
		rr := callUpdateMemberRole(t, db, orgCtrl, orgID, memberUserID, managerRoleID, ownerToken)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		newRole := getMemberRoleID(t, db, orgID, memberUserID)
		if newRole != managerRoleID {
			t.Errorf("expected member role to be Manager %s, got %s", managerRoleID, newRole)
		}
	})

	// Change back to User
	t.Run("Owner can change member role back to User", func(t *testing.T) {
		rr := callUpdateMemberRole(t, db, orgCtrl, orgID, memberUserID, userRoleID, ownerToken)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		newRole := getMemberRoleID(t, db, orgID, memberUserID)
		if newRole != userRoleID {
			t.Errorf("expected member role to be User %s, got %s", userRoleID, newRole)
		}
	})
}

// TestUpdateMemberRole_RegularMemberForbidden verifies that a regular member
// (User role) cannot change another member's role — gets 403 from PermissionMiddleware.
func TestUpdateMemberRole_RegularMemberForbidden(t *testing.T) {
	orgID, ownerToken, _, memberToken := setupOrgWithMembers(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	// Create a third user to be the target of the role change
	thirdUUID := utility.GenerateUUID()
	thirdSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("third_%v@qa.team", thirdUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Third",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("third_%v", thirdUUID),
	}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, thirdSignUp, false)
	thirdLogin := models.LoginRequestModel{Email: thirdSignUp.Email, Password: thirdSignUp.Password}
	thirdToken := tst.GetLoginToken(t, r, authCtrl, thirdLogin)
	thirdUserID := tst.GetUserIDFromToken(t, thirdToken, db)

	userRoleID := getRoleIDByName(t, db, "User")
	managerRoleID := getRoleIDByName(t, db, "Manager")
	addMemberViaEndpoint(t, db, orgCtrl, orgID, thirdUserID, userRoleID, ownerToken)

	// Regular member tries to change the third user's role — should get 403
	rr := callUpdateMemberRole(t, db, orgCtrl, orgID, thirdUserID, managerRoleID, memberToken)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for regular member, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify role was NOT changed
	currentRole := getMemberRoleID(t, db, orgID, thirdUserID)
	if currentRole != userRoleID {
		t.Errorf("expected third user role to remain User %s, got %s", userRoleID, currentRole)
	}
}

// TestUpdateMemberRole_NonCreatorAdminCanChangeRole verifies that an admin who
// is NOT the org owner can still change member roles (permission-based, not owner-only).
func TestUpdateMemberRole_NonCreatorAdminCanChangeRole(t *testing.T) {
	orgID, ownerToken, memberUserID, _ := setupOrgWithMembers(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	// Create a second admin (not the owner)
	adminUUID := utility.GenerateUUID()
	adminSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("admin2_%v@qa.team", adminUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Admin2",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("admin2_%v", adminUUID),
	}
	adminLogin := models.LoginRequestModel{Email: adminSignUp.Email, Password: adminSignUp.Password}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, adminSignUp, false)
	adminToken := tst.GetLoginToken(t, r, authCtrl, adminLogin)
	adminUserID := tst.GetUserIDFromToken(t, adminToken, db)

	adminRoleID := getRoleIDByName(t, db, "Administrator")
	addMemberViaEndpoint(t, db, orgCtrl, orgID, adminUserID, adminRoleID, ownerToken)

	// Switch admin's current org and re-login
	db.Postgresql.Model(&models.User{}).Where("id = ?", adminUserID).
		Update("current_org", orgID)
	r2 := gin.Default()
	adminToken = tst.GetLoginToken(t, r2, authCtrl, adminLogin)

	managerRoleID := getRoleIDByName(t, db, "Manager")

	// Non-owner admin changes member's role
	t.Run("Non-owner admin can change member role", func(t *testing.T) {
		rr := callUpdateMemberRole(t, db, orgCtrl, orgID, memberUserID, managerRoleID, adminToken)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		newRole := getMemberRoleID(t, db, orgID, memberUserID)
		if newRole != managerRoleID {
			t.Errorf("expected member role to be Manager %s, got %s", managerRoleID, newRole)
		}
	})
}

// TestUpdateMemberRole_InvalidRoleID verifies that providing a non-existent role_id
// returns an appropriate error (not 500).
func TestUpdateMemberRole_InvalidRoleID(t *testing.T) {
	orgID, ownerToken, memberUserID, _ := setupOrgWithMembers(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	fakeRoleID := utility.GenerateUUID()

	rr := callUpdateMemberRole(t, db, orgCtrl, orgID, memberUserID, fakeRoleID, ownerToken)
	if rr.Code == http.StatusOK {
		t.Errorf("expected error for non-existent role_id, got 200")
	}
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for non-existent role, got %d: %s", rr.Code, rr.Body.String())
	}
}
