package test_invitation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

const adminResendPath = "/api/v1/invite/admin-resend"

// adminResendRouter mirrors the production registration of POST
// /api/v1/invite/admin-resend, which is currently served without auth
// middleware.
func adminResendRouter(env *inviteEnv) *gin.Engine {
	r := gin.Default()
	r.POST(adminResendPath, env.InviteCtl.AdminResend)
	return r
}

func postAdminResend(t *testing.T, env *inviteEnv, body models.ResendCondition) *httptest.ResponseRecorder {
	t.Helper()

	var b bytes.Buffer
	assert.NoError(t, json.NewEncoder(&b).Encode(body))

	req, err := http.NewRequest(http.MethodPost, adminResendPath, &b)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	adminResendRouter(env).ServeHTTP(rr, req)
	return rr
}

// timeFromAWeekAgo keeps created_at inside the BETWEEN window for any
// invitation seeded during the test run.
func timeFromAWeekAgo() string {
	return time.Now().UTC().AddDate(0, 0, -7).Format("2006-01-02")
}

// uniqueDomain yields an email domain no other test can collide with, so the
// LIKE '%<extension>' predicate matches exactly the seeded invitation.
func uniqueDomain() string {
	return fmt.Sprintf("%v.example", utility.GenerateUUID())
}

// TestAdminResend_NoMatchesReturnsNotFound is the regression test for the
// reported panic: a valid time_from plus an extension matching no pending
// invitation must produce a controlled 404 instead of an index-out-of-range
// crash on invites[0].
func TestAdminResend_NoMatchesReturnsNotFound(t *testing.T) {
	env := newInviteEnv(t)

	body := models.ResendCondition{
		Extension: "@" + uniqueDomain(),
		TimeFrom:  timeFromAWeekAgo(),
	}

	rr := postAdminResend(t, env, body)
	tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)

	raw, _ := json.Marshal(tst.ParseResponse(rr))
	assert.True(t, strings.Contains(string(raw), "no pending invitations matched the supplied criteria"),
		"expected the no-match message in the response, got: %s", string(raw))
}

// TestAdminResend_MatchingInvitationUsesResendFlow proves the guard only
// intercepts the empty case: at least one pending invitation must still walk
// the original resend flow, including the 48h expiry refresh.
func TestAdminResend_MatchingInvitationUsesResendFlow(t *testing.T) {
	env := newInviteEnv(t)

	domain := uniqueDomain()
	email := fmt.Sprintf("resendable@%v", domain)
	token, err := utility.GenerateInvitationToken()
	assert.NoError(t, err)

	invite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          token,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: env.OrgID,
		InvitedBy:      tst.GetUserIDFromToken(t, env.Token, env.DB),
		ExpiresAt:      time.Now().Add(time.Hour).UTC(),
	}
	assert.NoError(t, env.DB.Postgresql.Create(&invite).Error)

	var before models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("id = ?", invite.ID).First(&before).Error)

	body := models.ResendCondition{
		Extension: "@" + domain,
		TimeFrom:  timeFromAWeekAgo(),
	}
	rr := postAdminResend(t, env, body)
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	var after models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("id = ?", invite.ID).First(&after).Error)
	assert.True(t, after.ExpiresAt.After(before.ExpiresAt),
		"matching invitation must still be refreshed by the resend flow; before=%v after=%v",
		before.ExpiresAt, after.ExpiresAt)
}

// TestAdminResend_InvalidTimeFromReturnsBadRequest guards the date parsing
// path: an unparseable time_from stays a controlled 400 and never reaches the
// invitation query.
func TestAdminResend_InvalidTimeFromReturnsBadRequest(t *testing.T) {
	env := newInviteEnv(t)

	body := models.ResendCondition{
		Extension: "@" + uniqueDomain(),
		TimeFrom:  "not-a-date",
	}

	rr := postAdminResend(t, env, body)
	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	raw, _ := json.Marshal(tst.ParseResponse(rr))
	assert.True(t, strings.Contains(string(raw), "error parsing the time into the right format"),
		"expected the date parsing error in the response, got: %s", string(raw))
}
