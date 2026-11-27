package test_organisation

import (
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
	tst "github.com/hngprojects/telex_be/tests"
)

// callPermissionProbe calls a route guarded only by Authorize and
// PermissionMiddleware(can_change_user_org_role). With orgParam set the route
// carries :org_id; otherwise the middleware falls back to the token's org.
func callPermissionProbe(t *testing.T, db *storage.Database, token, orgParam string) *httptest.ResponseRecorder {
	t.Helper()

	r := gin.Default()
	g := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
	perm := middleware.PermissionMiddleware(db.Postgresql, db.Redis, "can_change_user_org_role")
	g.GET("/probe", perm, ok)
	g.GET("/organisations/:org_id/probe", perm, ok)

	path := "/api/v1/probe"
	if orgParam != "" {
		path = fmt.Sprintf("/api/v1/organisations/%s/probe", orgParam)
	}
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// TestPermissionMiddleware_DemotedAdminLosesRoleImmediately checks that the
// middleware reads the caller's role from the database, not from the token's
// role_id claim, so a demotion takes effect on the very next request.
func TestPermissionMiddleware_DemotedAdminLosesRoleImmediately(t *testing.T) {
	orgID, ownerToken, memberUserID, _ := setupOrgWithMembers(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	authCtrl := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	adminRoleID := getRoleIDByName(t, db, "Administrator")
	userRoleID := getRoleIDByName(t, db, "User")

	// Promote the member, then log in so the new token carries the admin role_id.
	rr := callUpdateMemberRole(t, db, orgCtrl, orgID, memberUserID, adminRoleID, ownerToken)
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	// Switching current org copies the membership role into users.org_role_id
	// (services/user/user.go SwitchUserOrg), and login puts that value in the
	// token's role_id claim. UpdateMemberRole does not touch it, so it goes stale.
	if err := db.Postgresql.Model(&models.User{}).Where("id = ?", memberUserID).
		Update("org_role_id", adminRoleID).Error; err != nil {
		t.Fatalf("failed to set org_role_id: %v", err)
	}

	var member models.User
	if err := db.Postgresql.Where("id = ?", memberUserID).First(&member).Error; err != nil {
		t.Fatalf("failed to load member: %v", err)
	}
	adminToken := tst.GetLoginToken(t, gin.Default(), authCtrl,
		models.LoginRequestModel{Email: member.Email, Password: "password"})

	t.Run("admin is allowed", func(t *testing.T) {
		rr := callPermissionProbe(t, db, adminToken, "")
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		rr = callPermissionProbe(t, db, adminToken, orgID)
		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
	})

	// Demote through the real endpoint. The admin's token is still live.
	rr = callUpdateMemberRole(t, db, orgCtrl, orgID, memberUserID, userRoleID, ownerToken)
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	t.Run("demoted admin is denied with the same token", func(t *testing.T) {
		rr := callPermissionProbe(t, db, adminToken, "")
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
		rr = callPermissionProbe(t, db, adminToken, orgID)
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})

	t.Run("deactivated member is denied", func(t *testing.T) {
		// Re-promote, then deactivate: the role would allow it, the flag must not.
		if err := db.Postgresql.Model(&models.OrgUserManagement{}).
			Where("user_id = ? AND organisation_id = ?", memberUserID, orgID).
			Updates(map[string]any{"role_id": adminRoleID, "is_deactivated": true}).Error; err != nil {
			t.Fatalf("failed to update membership: %v", err)
		}
		rr := callPermissionProbe(t, db, adminToken, "")
		tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
	})

	t.Run("removed membership is denied", func(t *testing.T) {
		if err := db.Postgresql.Where("user_id = ? AND organisation_id = ?", memberUserID, orgID).
			Delete(&models.OrgUserManagement{}).Error; err != nil {
			t.Fatalf("failed to delete membership: %v", err)
		}
		rr := callPermissionProbe(t, db, adminToken, "")
		tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
	})
}

// TestPermissionMiddleware_OtherOrgDenied checks that an org admin is refused
// on another organisation's :org_id.
func TestPermissionMiddleware_OtherOrgDenied(t *testing.T) {
	_, ownerToken, _, _ := setupOrgWithMembers(t)
	otherOrgID, _, _, _ := setupOrgWithMembers(t)

	db := storage.Connection()
	rr := callPermissionProbe(t, db, ownerToken, otherOrgID)
	tst.AssertStatusCode(t, rr.Code, http.StatusForbidden)
}
