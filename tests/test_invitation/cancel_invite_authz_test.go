package test_invitation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func seedInviteBy(t *testing.T, env *inviteEnv, invitedBy string) string {
	t.Helper()
	tok, _ := utility.GenerateInvitationToken()
	invite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          fmt.Sprintf("cancel%v@example.com", utility.GenerateUUID()),
		Token:          tok,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: env.OrgID,
		InvitedBy:      invitedBy,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, env.DB.Postgresql.Create(&invite).Error)
	return invite.ID
}

func cancelAs(t *testing.T, env *inviteEnv, token, inviteID string) int {
	t.Helper()
	r := gin.Default()
	r.DELETE("/api/v1/invite/:invite_id", middleware.Authorize(env.DB.Postgresql), env.InviteCtl.CancelInvitation)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/invite/"+inviteID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr.Code
}

func inviteExists(t *testing.T, env *inviteEnv, id string) bool {
	t.Helper()
	var n int64
	assert.NoError(t, env.DB.Postgresql.Model(&models.Invitation{}).Where("id = ?", id).Count(&n).Error)
	return n > 0
}

func TestCancelInvitation_RequiresPermissionOrInviter(t *testing.T) {
	orgA := newInviteEnv(t)
	outsider := newInviteEnv(t) // owns a different organisation
	member := newInviteEnv(t)   // plain "User" member of orgA

	var userRole models.OrgRole
	assert.NoError(t, orgA.DB.Postgresql.Where("name = ?", models.OrgRoleNameUser).First(&userRole).Error)
	ownerID := tst.GetUserIDFromToken(t, orgA.Token, orgA.DB)
	memberID := tst.GetUserIDFromToken(t, member.Token, member.DB)
	assert.NoError(t, orgA.DB.Postgresql.Create(&models.OrgUserManagement{
		UserID: memberID, OrganisationID: orgA.OrgID, RoleID: userRole.ID, Status: "active",
	}).Error)

	// Outsider cannot cancel orgA's invite.
	ownerInvite := seedInviteBy(t, orgA, ownerID)
	tst.AssertStatusCode(t, cancelAs(t, orgA, outsider.Token, ownerInvite), http.StatusNotFound)
	assert.True(t, inviteExists(t, orgA, ownerInvite), "outsider must not delete the invite")

	// Plain member who did not send it cannot cancel.
	tst.AssertStatusCode(t, cancelAs(t, orgA, member.Token, ownerInvite), http.StatusNotFound)
	assert.True(t, inviteExists(t, orgA, ownerInvite), "plain member must not delete another's invite")

	// Plain member who sent it can cancel.
	memberInvite := seedInviteBy(t, orgA, memberID)
	tst.AssertStatusCode(t, cancelAs(t, orgA, member.Token, memberInvite), http.StatusOK)
	assert.False(t, inviteExists(t, orgA, memberInvite))

	// Inviter who has been deactivated can no longer cancel.
	deactivatedInvite := seedInviteBy(t, orgA, memberID)
	assert.NoError(t, orgA.DB.Postgresql.Model(&models.OrgUserManagement{}).
		Where("user_id = ? AND organisation_id = ?", memberID, orgA.OrgID).
		Update("is_deactivated", true).Error)
	tst.AssertStatusCode(t, cancelAs(t, orgA, member.Token, deactivatedInvite), http.StatusNotFound)
	assert.True(t, inviteExists(t, orgA, deactivatedInvite))

	// Owner can cancel.
	tst.AssertStatusCode(t, cancelAs(t, orgA, orgA.Token, ownerInvite), http.StatusOK)
	assert.False(t, inviteExists(t, orgA, ownerInvite))

	// Unknown id is 404 (was 500); malformed id stays 400.
	tst.AssertStatusCode(t, cancelAs(t, orgA, orgA.Token, utility.GenerateUUID()), http.StatusNotFound)
	tst.AssertStatusCode(t, cancelAs(t, orgA, orgA.Token, "not-a-uuid"), http.StatusBadRequest)
}
