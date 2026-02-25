package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// setupTwoUsersInOrg creates an org with an admin and a second member, returns
// orgId, adminToken, memberUserId, memberToken.
func setupTwoUsersInOrg(t *testing.T) (string, string, string, string) {
	t.Helper()

	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	// --- admin user ---
	adminUUID := utility.GenerateUUID()
	adminSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("admin%v@qa.team", adminUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Admin",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("admin_%v", adminUUID),
	}
	adminLogin := models.LoginRequestModel{
		Email:    adminSignUp.Email,
		Password: adminSignUp.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, adminSignUp, false)
	adminToken := tst.GetLoginToken(t, r, authCtrl, adminLogin)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("StatusTestOrg%s", adminUUID),
		Description: "Org for testing status change",
		Email:       adminSignUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, adminToken)

	// --- member user (use a fresh router to avoid duplicate route registration) ---
	memberUUID := utility.GenerateUUID()
	memberSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("member%v@qa.team", memberUUID),
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

	r2 := gin.Default()
	tst.SignupUser(t, r2, authCtrl, memberSignUp, false)
	memberToken := tst.GetLoginToken(t, r2, authCtrl, memberLogin)
	memberUserID := tst.GetUserIDFromToken(t, memberToken, db)

	// Add member to organisation (use yet another fresh router)
	r3 := gin.Default()
	addMemberUrl := r3.Group("/api/v1", middleware.Authorize(db.Postgresql))
	addMemberUrl.POST("/organisations/:org_id/members", orgCtrl.AddMemberToOrganisation)

	addBody := models.OrgUserCreateRequest{
		UserID: memberUserID,
		RoleID: getDefaultRoleID(t, db),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(addBody)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisations/%s/members", orgId), &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr := httptest.NewRecorder()
	r3.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Logf("Warning: Add member returned status %d: %s", rr.Code, rr.Body.String())
	}

	return orgId, adminToken, memberUserID, memberToken
}

// getDefaultRoleID returns the ID of the "User" org role.
func getDefaultRoleID(t *testing.T, db *storage.Database) string {
	t.Helper()
	var role models.OrgRole
	err := db.Postgresql.Where("name = ?", "User").First(&role).Error
	if err != nil {
		t.Fatalf("failed to get default User role: %v", err)
	}
	return role.ID
}

// helper to call PATCH /:org_id/users/:user_id/status and return the response
func callChangeStatus(t *testing.T, db *storage.Database, orgCtrl organisation.Controller, orgID, userID, token string, active bool) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.Default()
	orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	orgUrl.PATCH("/:org_id/users/:user_id/status", orgCtrl.ChangeMemberActiveStatus)

	body := map[string]bool{"active": active}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	reqURI := url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/users/%s/status", orgID, userID)}
	req, err := http.NewRequest(http.MethodPatch, reqURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// helper to read org_user_managements row and return is_deactivated, status
func getOrgMemberState(t *testing.T, db *storage.Database, userID, orgID string) (bool, string) {
	t.Helper()
	var oum models.OrgUserManagement
	err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", userID, orgID).First(&oum).Error
	if err != nil {
		t.Fatalf("failed to query OrgUserManagement: %v", err)
	}
	return oum.IsDeactivated, oum.Status
}

func TestChangeMemberActiveStatus(t *testing.T) {
	orgId, adminToken, memberUserID, _ := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	// 1. Deactivate the member
	t.Run("Deactivate member", func(t *testing.T) {
		rr := callChangeStatus(t, db, orgCtrl, orgId, memberUserID, adminToken, false)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		tst.AssertResponseMessage(t, data["message"].(string), "success")

		// verify the response says "deactivated"
		dataStr, ok := data["data"].(string)
		if !ok || dataStr != "user deactivated successfully" {
			t.Errorf("expected response data 'user deactivated successfully', got %q", dataStr)
		}
	})

	// 2. Verify DB state after deactivation
	t.Run("Verify DB state after deactivation", func(t *testing.T) {
		isDeactivated, status := getOrgMemberState(t, db, memberUserID, orgId)
		if !isDeactivated {
			t.Errorf("expected is_deactivated=true after deactivation, got %v", isDeactivated)
		}
		if status != "inactive" {
			t.Errorf("expected status='inactive' after deactivation, got %q", status)
		}
	})

	// 3. Reactivate the member
	t.Run("Reactivate member", func(t *testing.T) {
		rr := callChangeStatus(t, db, orgCtrl, orgId, memberUserID, adminToken, true)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		tst.AssertResponseMessage(t, data["message"].(string), "success")

		// verify the response says "activated"
		dataStr, ok := data["data"].(string)
		if !ok || dataStr != "user activated successfully" {
			t.Errorf("expected response data 'user activated successfully', got %q", dataStr)
		}
	})

	// 4. Verify DB state after reactivation
	t.Run("Verify DB state after reactivation", func(t *testing.T) {
		isDeactivated, status := getOrgMemberState(t, db, memberUserID, orgId)
		if isDeactivated {
			t.Errorf("BUG: is_deactivated still true after reactivation")
		}
		if status != "active" {
			t.Errorf("BUG: status=%q after reactivation, expected 'active'", status)
		}
	})

	// 5. Deactivate again to confirm toggle works repeatedly
	t.Run("Deactivate again after reactivation", func(t *testing.T) {
		rr := callChangeStatus(t, db, orgCtrl, orgId, memberUserID, adminToken, false)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		isDeactivated, status := getOrgMemberState(t, db, memberUserID, orgId)
		if !isDeactivated {
			t.Errorf("expected is_deactivated=true after second deactivation, got %v", isDeactivated)
		}
		if status != "inactive" {
			t.Errorf("expected status='inactive' after second deactivation, got %q", status)
		}
	})

	// 6. Reactivate again to confirm full cycle
	t.Run("Reactivate again after second deactivation", func(t *testing.T) {
		rr := callChangeStatus(t, db, orgCtrl, orgId, memberUserID, adminToken, true)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		isDeactivated, status := getOrgMemberState(t, db, memberUserID, orgId)
		if isDeactivated {
			t.Errorf("BUG: is_deactivated still true after second reactivation")
		}
		if status != "active" {
			t.Errorf("BUG: status=%q after second reactivation, expected 'active'", status)
		}
	})
}

func TestChangeMemberActiveStatus_SelfChange(t *testing.T) {
	_, adminToken, _, _ := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()
	adminUserID := tst.GetUserIDFromToken(t, adminToken, db)

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	rr := callChangeStatus(t, db, orgCtrl, utility.GenerateUUID(), adminUserID, adminToken, false)
	tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
}

func TestChangeMemberActiveStatus_NonAdmin(t *testing.T) {
	orgId, _, memberUserID, memberToken := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	// a non-admin member should not be able to deactivate another member
	rr := callChangeStatus(t, db, orgCtrl, orgId, memberUserID, memberToken, false)
	tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
}

func TestChangeMemberActiveStatus_InvalidUserID(t *testing.T) {
	orgId, adminToken, _, _ := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	rr := callChangeStatus(t, db, orgCtrl, orgId, "not-a-uuid", adminToken, false)
	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)
}

func TestChangeMemberActiveStatus_NonExistentUser(t *testing.T) {
	orgId, adminToken, _, _ := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	rr := callChangeStatus(t, db, orgCtrl, orgId, utility.GenerateUUID(), adminToken, false)
	if rr.Code == http.StatusOK {
		t.Errorf("expected error status for non-existent user, got 200")
	}
}
