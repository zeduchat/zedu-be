package test_invitation

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// seedPendingInvite stores a pending invitation in env's organisation and
// returns its email and token.
func seedPendingInvite(t *testing.T, env *inviteEnv) (string, string) {
	t.Helper()
	email := fmt.Sprintf("pending%v@example.com", utility.GenerateUUID())
	tok, _ := utility.GenerateInvitationToken()
	invite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          tok,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: env.OrgID,
		InvitedBy:      tst.GetUserIDFromToken(t, env.Token, env.DB),
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, env.DB.Postgresql.Create(&invite).Error)
	return email, tok
}

func storedInviteToken(t *testing.T, env *inviteEnv, email string) string {
	t.Helper()
	var inv models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("email = ? AND organisation_id = ?", email, env.OrgID).First(&inv).Error)
	return inv.Token
}

// resendAs calls POST /invite/resend on env's router with another user's token.
func resendAs(t *testing.T, env *inviteEnv, token string, body models.ResendInvitationRequest) (int, string) {
	t.Helper()
	caller := *env
	caller.Token = token
	rr := caller.doJSON(t, http.MethodPost, "/api/v1/invite/resend", body, true)
	return rr.Code, rr.Body.String()
}

func TestResendEndpoint_RequiresInvitePermissionInOrg(t *testing.T) {
	orgA := newInviteEnv(t)
	outsider := newInviteEnv(t) // owns a different organisation
	member := newInviteEnv(t)   // will be a plain "User" member of orgA

	var userRole models.OrgRole
	assert.NoError(t, orgA.DB.Postgresql.Where("name = ?", models.OrgRoleNameUser).First(&userRole).Error)
	memberID := tst.GetUserIDFromToken(t, member.Token, member.DB)
	assert.NoError(t, orgA.DB.Postgresql.Create(&models.OrgUserManagement{
		UserID: memberID, OrganisationID: orgA.OrgID, RoleID: userRole.ID, Status: "active",
	}).Error)

	email, original := seedPendingInvite(t, orgA)
	body := models.ResendInvitationRequest{Email: email, OrganisationID: orgA.OrgID}

	// Outsider: 404, token unchanged.
	code, outsiderBody := resendAs(t, orgA, outsider.Token, body)
	tst.AssertStatusCode(t, code, http.StatusNotFound)
	assert.Equal(t, original, storedInviteToken(t, orgA, email), "outsider must not rotate the token")

	// Outsider probing an email with no invitation gets the identical response.
	ghost := models.ResendInvitationRequest{Email: fmt.Sprintf("ghost%v@example.com", utility.GenerateUUID()), OrganisationID: orgA.OrgID}
	code, ghostBody := resendAs(t, orgA, outsider.Token, ghost)
	tst.AssertStatusCode(t, code, http.StatusNotFound)
	assert.Equal(t, outsiderBody, ghostBody, "no-invite and not-allowed must look the same")

	// Outsider naming a nonexistent organisation also gets the same response.
	missingOrg := models.ResendInvitationRequest{Email: email, OrganisationID: utility.GenerateUUID()}
	code, missingBody := resendAs(t, orgA, outsider.Token, missingOrg)
	tst.AssertStatusCode(t, code, http.StatusNotFound)
	assert.Equal(t, outsiderBody, missingBody)

	// Plain member without can_invite_members: 404, token unchanged.
	code, _ = resendAs(t, orgA, member.Token, body)
	tst.AssertStatusCode(t, code, http.StatusNotFound)
	assert.Equal(t, original, storedInviteToken(t, orgA, email), "plain member must not rotate the token")

	// Owner: 200 and the token rotates.
	code, _ = resendAs(t, orgA, orgA.Token, body)
	tst.AssertStatusCode(t, code, http.StatusOK)
	assert.NotEqual(t, original, storedInviteToken(t, orgA, email), "owner resend must rotate the token")
}
