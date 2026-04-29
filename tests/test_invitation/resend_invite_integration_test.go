package test_invitation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// inviteEnv holds a freshly-provisioned inviter (with login token) and an
// organisation, plus controller instances wired to the test DB.
type inviteEnv struct {
	DB        *storage.Database
	OrgID     string
	Token     string
	UserEmail string
	InviteCtl invitation.Controller
}

func newInviteEnv(t *testing.T) *inviteEnv {
	t.Helper()

	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	v := validator.New()
	db := storage.Connection()

	uid := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("inviter%v@qa.team", uid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "inviter",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("inviter_%v", uid),
	}
	loginReq := models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password}

	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, signUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, loginReq)

	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger}
	createOrg := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("org%v", uid),
		Description: "integration env org",
		Email:       signUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrg, token)

	return &inviteEnv{
		DB:        db,
		OrgID:     orgID,
		Token:     token,
		UserEmail: signUp.Email,
		InviteCtl: invitation.Controller{Db: db, Validator: v, Logger: logger,
			ExtReq: request.ExternalRequest{Logger: logger, Test: true}},
	}
}

func (e *inviteEnv) router() *gin.Engine {
	r := gin.Default()
	g := r.Group("/api/v1/invite")
	g.POST("", middleware.Authorize(e.DB.Postgresql), e.InviteCtl.OrganisationInviteMany)
	g.POST("/resend", middleware.Authorize(e.DB.Postgresql), e.InviteCtl.ResendInvitation)
	g.POST("/verify", e.InviteCtl.OrganisationVerifyInvite)
	return r
}

func (e *inviteEnv) doJSON(t *testing.T, method, path string, body any, withAuth bool) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	assert.NoError(t, json.NewEncoder(&b).Encode(body))
	req, err := http.NewRequest(method, (&url.URL{Path: path}).String(), &b)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if withAuth {
		req.Header.Set("Authorization", "Bearer "+e.Token)
	}
	rr := httptest.NewRecorder()
	e.router().ServeHTTP(rr, req)
	return rr
}

// TestInviteEndpoint_ResendsWhenAlreadyInvited is the regression test for the
// reported bug: re-POSTing to /api/v1/invite for an email that already has a
// pending invitation must succeed and the stored token must rotate (so the new
// email's link is fresh and valid).
func TestInviteEndpoint_ResendsWhenAlreadyInvited(t *testing.T) {
	env := newInviteEnv(t)
	email := fmt.Sprintf("dup%v@example.com", utility.GenerateUUID())

	body := models.InvitationCreateReq{
		Emails:         []string{email},
		OrganisationID: env.OrgID,
		RoleID:         "00000000-0000-0000-0000-000000000000",
	}

	// First invite — should succeed, persists a row.
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite", body, true)
	tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

	var first models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("email = ? AND organisation_id = ?", email, env.OrgID).First(&first).Error)
	originalToken := first.Token

	// Sleep so created_at delta is observable.
	time.Sleep(1100 * time.Millisecond)

	// Second invite for the same email — must NOT error, must rotate the token.
	rr = env.doJSON(t, http.MethodPost, "/api/v1/invite", body, true)
	tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

	var rows []models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("email = ? AND organisation_id = ?", email, env.OrgID).Find(&rows).Error)
	assert.Len(t, rows, 1, "re-invite must update in place")

	after := rows[0]
	assert.NotEqual(t, originalToken, after.Token, "re-invite must rotate the token")
	assert.True(t, after.CreatedAt.After(first.CreatedAt), "re-invite must bump created_at")
}

// TestInviteEndpoint_AlreadyMemberClearError verifies the controller now
// surfaces the actual reason in the error payload (Fix 3) instead of always
// blaming "pending invitations".
func TestInviteEndpoint_AlreadyMemberClearError(t *testing.T) {
	env := newInviteEnv(t)

	// The inviter is already a member of their own org; inviting their email
	// triggers the "already a member" branch.
	body := models.InvitationCreateReq{
		Emails:         []string{env.UserEmail},
		OrganisationID: env.OrgID,
		RoleID:         "00000000-0000-0000-0000-000000000000",
	}
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite", body, true)
	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	resp := tst.ParseResponse(rr)
	msg, _ := resp["message"].(string)
	assert.False(t, strings.Contains(msg, "pending invitation"),
		"controller must not blame 'pending invitation' for member case; got %q", msg)

	// The per-email error string lives in the response's "error" field.
	errs, _ := resp["error"].([]any)
	assert.NotEmpty(t, errs, "expected the underlying per-email error to be surfaced")
	if len(errs) > 0 {
		first, _ := errs[0].(string)
		assert.True(t, strings.Contains(first, "already a member"),
			"expected 'already a member' in errs[0]; got %q", first)
	}
}

// TestResendEndpoint_NoInvitation verifies Fix 4 via HTTP: resending for an
// email that was never invited returns "no invitation found", not the
// misleading "already accepted".
func TestResendEndpoint_NoInvitation(t *testing.T) {
	env := newInviteEnv(t)

	body := models.ResendInvitationRequest{
		Email:          fmt.Sprintf("ghost%v@example.com", utility.GenerateUUID()),
		OrganisationID: env.OrgID,
	}
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite/resend", body, true)
	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	resp := tst.ParseResponse(rr)
	// The service-level error is wrapped into the response's "error" field
	// by the controller. We assert on the stringified payload to be tolerant
	// of how it's serialised.
	raw, _ := json.Marshal(resp)
	assert.True(t, strings.Contains(string(raw), "no invitation found"),
		"expected 'no invitation found' in response, got: %s", string(raw))
}

// TestResendEndpoint_AlreadyAccepted verifies the resend controller still
// returns the "already accepted" error when the invitation row exists with
// status='accepted'.
func TestResendEndpoint_AlreadyAccepted(t *testing.T) {
	env := newInviteEnv(t)

	// Seed an accepted invitation directly.
	email := fmt.Sprintf("accepted%v@example.com", utility.GenerateUUID())
	tok, _ := utility.GenerateInvitationToken()
	invite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          tok,
		Status:         "accepted",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: env.OrgID,
		InvitedBy:      tst.GetUserIDFromToken(t, env.Token, env.DB),
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, env.DB.Postgresql.Create(&invite).Error)

	body := models.ResendInvitationRequest{Email: email, OrganisationID: env.OrgID}
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite/resend", body, true)
	tst.AssertStatusCode(t, rr.Code, http.StatusBadRequest)

	raw, _ := json.Marshal(tst.ParseResponse(rr))
	assert.True(t, strings.Contains(string(raw), "already accepted"),
		"expected 'already accepted' in response, got: %s", string(raw))
}
