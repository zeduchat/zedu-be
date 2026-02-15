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

func TestChangeMemberActiveStatus(t *testing.T) {
	orgId, adminToken, memberUserID, _ := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	tests := []struct {
		Name         string
		RequestBody  models.ChangeMemberActiveStatus
		ExpectedCode int
		Message      string
		UserID       string
		OrgID        string
		Token        string
	}{
		{
			Name:         "Deactivate member - should succeed",
			RequestBody:  models.ChangeMemberActiveStatus{Activate: false},
			ExpectedCode: http.StatusOK,
			Message:      "success",
			UserID:       memberUserID,
			OrgID:        orgId,
			Token:        adminToken,
		},
		{
			Name:         "Reactivate member - should succeed and actually update",
			RequestBody:  models.ChangeMemberActiveStatus{Activate: true},
			ExpectedCode: http.StatusOK,
			Message:      "success",
			UserID:       memberUserID,
			OrgID:        orgId,
			Token:        adminToken,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			r := gin.Default()
			orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
			orgUrl.PATCH("/:org_id/users/:user_id/status", orgCtrl.ChangeMemberActiveStatus)

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			reqURI := url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/users/%s/status", test.OrgID, test.UserID)}
			req, err := http.NewRequest(http.MethodPatch, reqURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+test.Token)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)
			data := tst.ParseResponse(rr)
			code := int(data["status_code"].(float64))
			tst.AssertStatusCode(t, code, test.ExpectedCode)

			if test.Message != "" {
				message := data["message"]
				if message != nil {
					tst.AssertResponseMessage(t, message.(string), test.Message)
				}
			}
		})
	}

	// --- Critical assertion: verify the DB state after reactivation ---
	t.Run("Verify member is_deactivated=false after reactivation", func(t *testing.T) {
		var oum models.OrgUserManagement
		err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", memberUserID, orgId).First(&oum).Error
		if err != nil {
			t.Fatalf("failed to query OrgUserManagement: %v", err)
		}
		if oum.IsDeactivated {
			t.Errorf("BUG: member is still deactivated after reactivation request. is_deactivated=%v, expected false", oum.IsDeactivated)
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

	r := gin.Default()
	orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	orgUrl.PATCH("/:org_id/users/:user_id/status", orgCtrl.ChangeMemberActiveStatus)

	// We use any orgId here since the self-check should fail before DB lookup
	reqBody := models.ChangeMemberActiveStatus{Activate: false}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(reqBody)

	// Use a dummy orgId - the self-check triggers before org validation
	req, _ := http.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/v1/organisations/%s/users/%s/status", utility.GenerateUUID(), adminUserID), &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
}
