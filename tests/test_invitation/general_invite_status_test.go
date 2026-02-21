package test_invitation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	inviteCtrl "github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// setupOrgWithGeneralInvite creates a user, org, and a general (shareable) invite.
// Returns orgID, userToken, roleID.
func setupOrgWithGeneralInvite(t *testing.T) (string, string, string) {
	t.Helper()

	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	uuid := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("invtest%v@qa.team", uuid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "InvTest",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("invtest_%v", uuid),
	}
	login := models.LoginRequestModel{
		Email:    signUp.Email,
		Password: signUp.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, signUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, login)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("InvStatusOrg%s", uuid),
		Description: "Org for testing general invite status",
		Email:       signUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	// Re-login to get a fresh token with the correct org_id embedded
	// (CreateOrganisation updates user.CurrentOrg via the Authorize middleware)
	r2 := gin.Default()
	token = tst.GetLoginToken(t, r2, authCtrl, login)

	// Get a role ID for creating the shareable invite
	roleID := getDefaultRole(t, db)

	// Create a general invitation via the endpoint
	invCtrl := inviteCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r3 := gin.Default()
	invUrl := r3.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general", invCtrl.GeneralInvitationCreate)

	createBody := models.ShareableInviteRequest{
		OrganisationID: orgID,
		RoleID:         roleID,
	}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(createBody)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/invite/general", &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r3.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("Failed to create general invite: status %d, body: %s", rr.Code, rr.Body.String())
	}

	return orgID, token, roleID
}

func getDefaultRole(t *testing.T, db *storage.Database) string {
	t.Helper()
	var role models.OrgRole
	err := db.Postgresql.Where("name = ?", "User").First(&role).Error
	if err != nil {
		t.Fatalf("failed to get default User role: %v", err)
	}
	return role.ID
}

// callChangeGeneralInviteStatus calls POST /invite/general/change-status
func callChangeGeneralInviteStatus(t *testing.T, db *storage.Database, invCtrl inviteCtrl.Controller, token string, status bool) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/change-status", invCtrl.ChangeGeneralInviteStatus)

	body := map[string]bool{"status": status}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	reqURI := url.URL{Path: "/api/v1/invite/general/change-status"}
	req, err := http.NewRequest(http.MethodPost, reqURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// getGeneralInviteStatus reads the active_status of the most recent invite for a given org
func getGeneralInviteStatus(t *testing.T, db *storage.Database, orgID string) bool {
	t.Helper()
	var invite models.GeneralInvitation
	err := db.Postgresql.Where("organisation_id = ?", orgID).Order("created_at DESC").First(&invite).Error
	if err != nil {
		t.Fatalf("failed to query GeneralInvitation: %v", err)
	}
	return invite.ActiveStatus
}

// getMostRecentGeneralInvite returns the most recent GeneralInvitation for an org
func getMostRecentGeneralInvite(t *testing.T, db *storage.Database, orgID string) models.GeneralInvitation {
	t.Helper()
	var invite models.GeneralInvitation
	err := db.Postgresql.Where("organisation_id = ?", orgID).Order("created_at DESC").First(&invite).Error
	if err != nil {
		t.Fatalf("failed to query GeneralInvitation: %v", err)
	}
	return invite
}

// getGeneralInviteStatusByID reads the active_status of a specific invite by ID
func getGeneralInviteStatusByID(t *testing.T, db *storage.Database, inviteID string) bool {
	t.Helper()
	var invite models.GeneralInvitation
	err := db.Postgresql.Where("id = ?", inviteID).First(&invite).Error
	if err != nil {
		t.Fatalf("failed to query GeneralInvitation by ID: %v", err)
	}
	return invite.ActiveStatus
}

// addMemberToOrg adds a user to an org with a given role, using the admin token
func addMemberToOrg(t *testing.T, db *storage.Database, orgCtrl organisation.Controller, orgID, memberUserID, roleID, adminToken string) {
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

// getAdminRoleID returns the ID of the "Administrator" org role
func getAdminRoleID(t *testing.T, db *storage.Database) string {
	t.Helper()
	var role models.OrgRole
	err := db.Postgresql.Where("name = ?", "Administrator").First(&role).Error
	if err != nil {
		t.Fatalf("failed to get Administrator role: %v", err)
	}
	return role.ID
}


func TestChangeGeneralInviteStatus_DisableAndEnable(t *testing.T) {
	orgID, token, _ := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// 1. Verify initial state: active_status should be true
	t.Run("Initial state is active", func(t *testing.T) {
		active := getGeneralInviteStatus(t, db, orgID)
		if !active {
			t.Errorf("expected active_status=true after creation, got %v", active)
		}
	})

	// 2. Disable the invite link
	t.Run("Disable invite link", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, false)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		tst.AssertResponseMessage(t, data["message"].(string), "invitation updated successfully")
	})

	// 3. Verify DB state after disabling
	t.Run("Verify DB state after disabling", func(t *testing.T) {
		active := getGeneralInviteStatus(t, db, orgID)
		if active {
			t.Errorf("expected active_status=false after disabling, got %v", active)
		}
	})

	// 4. Re-enable the invite link (this was the bug — used to fail with "No active invitation found")
	t.Run("Re-enable invite link", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, true)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		tst.AssertResponseMessage(t, data["message"].(string), "invitation updated successfully")
	})

	// 5. Verify DB state after re-enabling
	t.Run("Verify DB state after re-enabling", func(t *testing.T) {
		active := getGeneralInviteStatus(t, db, orgID)
		if !active {
			t.Errorf("expected active_status=true after re-enabling, got %v", active)
		}
	})

	// 6. Disable again to confirm toggle works repeatedly
	t.Run("Disable again after re-enable", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, false)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		active := getGeneralInviteStatus(t, db, orgID)
		if active {
			t.Errorf("expected active_status=false after second disable, got %v", active)
		}
	})

	// 7. Re-enable again to confirm full cycle
	t.Run("Re-enable again after second disable", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, true)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		active := getGeneralInviteStatus(t, db, orgID)
		if !active {
			t.Errorf("expected active_status=true after second re-enable, got %v", active)
		}
	})
}

func TestChangeGeneralInviteStatus_NoInviteForOrg(t *testing.T) {
	// Create a user and org but do NOT create a general invite
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	uuid := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("noinv%v@qa.team", uuid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "NoInv",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("noinv_%v", uuid),
	}
	login := models.LoginRequestModel{
		Email:    signUp.Email,
		Password: signUp.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, signUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, login)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("NoInvOrg%s", uuid),
		Description: "Org with no general invite",
		Email:       signUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, token)

	// Re-login to get a fresh token with the correct org_id
	r2 := gin.Default()
	token = tst.GetLoginToken(t, r2, authCtrl, login)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// Trying to change status for an org with no invite should fail
	rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, false)
	if rr.Code == http.StatusOK {
		t.Errorf("expected error status for org with no general invite, got 200")
	}
}

// TestChangeGeneralInviteStatus_MultipleInvites verifies that when multiple
// invitations exist for the same org (e.g. an old expired one + a new one),
// change-status only operates on the most recent invitation.
func TestChangeGeneralInviteStatus_MultipleInvites(t *testing.T) {
	orgID, token, roleID := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()

	// Record the first (current) invite
	firstInvite := getMostRecentGeneralInvite(t, db, orgID)

	// Simulate an expired old invite by inserting one with an old created_at and expired expires_at
	oldInvite := models.GeneralInvitation{
		ID:             utility.GenerateUUID(),
		Token:          "old-expired-token",
		ActiveStatus:   true,
		Role:           roleID,
		OrganisationID: orgID,
		InvitedBy:      utility.GenerateUUID(), // some other user created this old link
		CreatedAt:      time.Now().Add(-72 * time.Hour),
		ExpiresAt:      time.Now().Add(-24 * time.Hour), // expired
	}
	if err := db.Postgresql.Create(&oldInvite).Error; err != nil {
		t.Fatalf("failed to insert old invite: %v", err)
	}

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// Disable the invite link — should target the most recent (firstInvite), not the old one
	t.Run("Disable targets most recent invite", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, false)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		// The most recent invite should be disabled
		recentActive := getGeneralInviteStatusByID(t, db, firstInvite.ID)
		if recentActive {
			t.Errorf("expected most recent invite active_status=false, got true")
		}

		// The old invite should be untouched
		oldActive := getGeneralInviteStatusByID(t, db, oldInvite.ID)
		if !oldActive {
			t.Errorf("expected old invite active_status to remain true (untouched), got false")
		}
	})

	// Re-enable — should also target only the most recent invite
	t.Run("Re-enable targets most recent invite", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, token, true)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		recentActive := getGeneralInviteStatusByID(t, db, firstInvite.ID)
		if !recentActive {
			t.Errorf("expected most recent invite active_status=true after re-enable, got false")
		}
	})
}

// TestChangeGeneralInviteStatus_NonCreatorAdmin verifies that an admin who did NOT
// create the invite link can still toggle it (admin/owner check, not InvitedBy).
func TestChangeGeneralInviteStatus_NonCreatorAdmin(t *testing.T) {
	// Setup: owner creates org + invite link
	orgID, ownerToken, _ := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	// Create a second user and add them as Administrator to the org
	secondUUID := utility.GenerateUUID()
	secondSignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("admin2_%v@qa.team", secondUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Admin2",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("admin2_%v", secondUUID),
	}
	secondLogin := models.LoginRequestModel{
		Email:    secondSignUp.Email,
		Password: secondSignUp.Password,
	}

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, secondSignUp, false)
	secondToken := tst.GetLoginToken(t, r, authCtrl, secondLogin)
	secondUserID := tst.GetUserIDFromToken(t, secondToken, db)

	// Add second user as Administrator
	adminRoleID := getAdminRoleID(t, db)
	addMemberToOrg(t, db, orgCtrl, orgID, secondUserID, adminRoleID, ownerToken)

	// Switch second user's current org so their token contains the right org_id
	db.Postgresql.Model(&models.User{}).Where("id = ?", secondUserID).
		Update("current_org", orgID)
	r2 := gin.Default()
	secondToken = tst.GetLoginToken(t, r2, authCtrl, secondLogin)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// The second admin (non-creator) should be able to disable the invite
	t.Run("Non-creator admin can disable invite", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, secondToken, false)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		active := getGeneralInviteStatus(t, db, orgID)
		if active {
			t.Errorf("expected active_status=false after non-creator admin disabled it, got true")
		}
	})

	// The second admin should also be able to re-enable it
	t.Run("Non-creator admin can re-enable invite", func(t *testing.T) {
		rr := callChangeGeneralInviteStatus(t, db, invCtrl, secondToken, true)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		active := getGeneralInviteStatus(t, db, orgID)
		if !active {
			t.Errorf("expected active_status=true after non-creator admin re-enabled it, got false")
		}
	})
}

// TestChangeGeneralInviteStatus_NonAdminMemberForbidden verifies that a regular
// member (non-admin, non-owner) cannot toggle the invite link.
func TestChangeGeneralInviteStatus_NonAdminMemberForbidden(t *testing.T) {
	orgID, ownerToken, _ := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	// Create a regular member
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

	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, memberSignUp, false)
	memberToken := tst.GetLoginToken(t, r, authCtrl, memberLogin)
	memberUserID := tst.GetUserIDFromToken(t, memberToken, db)

	// Add as regular "User" role member
	userRoleID := getDefaultRole(t, db)
	addMemberToOrg(t, db, orgCtrl, orgID, memberUserID, userRoleID, ownerToken)

	// Switch member's current org
	db.Postgresql.Model(&models.User{}).Where("id = ?", memberUserID).
		Update("current_org", orgID)
	r2 := gin.Default()
	memberToken = tst.GetLoginToken(t, r2, authCtrl, memberLogin)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// Regular member should be forbidden from changing invite status
	rr := callChangeGeneralInviteStatus(t, db, invCtrl, memberToken, false)
	if rr.Code == http.StatusOK {
		t.Errorf("expected non-admin member to be forbidden, got 200")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for non-admin member, got %d", rr.Code)
	}

	// Verify invite was NOT changed
	active := getGeneralInviteStatus(t, db, orgID)
	if !active {
		t.Errorf("invite should still be active after non-admin attempt, got false")
	}
}
