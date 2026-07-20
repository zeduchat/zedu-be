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

// TestInviteEndpoint_AlreadyMemberSucceeds verifies that re-inviting an
// existing member through the public endpoint now succeeds (instead of
// returning 400 with a misleading message). The acceptance flow handles the
// "already a member" case downstream.
func TestInviteEndpoint_AlreadyMemberSucceeds(t *testing.T) {
	env := newInviteEnv(t)

	// The inviter is already a member of their own org.
	body := models.InvitationCreateReq{
		Emails:         []string{env.UserEmail},
		OrganisationID: env.OrgID,
		RoleID:         "00000000-0000-0000-0000-000000000000",
	}
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite", body, true)
	tst.AssertStatusCode(t, rr.Code, http.StatusCreated)

	// And an invitation row should be persisted so the email link works.
	var invite models.Invitation
	assert.NoError(t, env.DB.Postgresql.
		Where("email = ? AND organisation_id = ?", env.UserEmail, env.OrgID).
		First(&invite).Error)
	assert.NotEmpty(t, invite.Token)
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

// TestVerifyEndpoint_AcceptingAsExistingMemberSucceeds verifies that when the
// recipient of an invite is already a member of the org, hitting /verify
// resolves cleanly: 200, status flips to "accepted", and the user's
// membership row is preserved untouched.
func TestVerifyEndpoint_AcceptingAsExistingMemberSucceeds(t *testing.T) {
	env := newInviteEnv(t)

	// Inviter is already a member of env.OrgID. Seed a fresh invitation
	// directly so we have a known token to verify with.
	tok, _ := utility.GenerateInvitationToken()
	invite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          env.UserEmail,
		Token:          tok,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: env.OrgID,
		InvitedBy:      tst.GetUserIDFromToken(t, env.Token, env.DB),
		IsTelexUser:    true,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, env.DB.Postgresql.Create(&invite).Error)

	// Snapshot the existing membership row so we can compare afterward.
	var beforeMember models.OrgUserManagement
	assert.NoError(t, env.DB.Postgresql.
		Where("user_id = ? AND organisation_id = ?", invite.InvitedBy, env.OrgID).
		First(&beforeMember).Error)

	body := models.VerifyInvitationLinkRequest{Token: tok}
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite/verify", body, false)
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	// Invitation should now be marked accepted.
	var afterInvite models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("id = ?", invite.ID).First(&afterInvite).Error)
	assert.Equal(t, "accepted", afterInvite.Status)

	// Membership row must be preserved (no duplicate row, role unchanged).
	var memberRows []models.OrgUserManagement
	assert.NoError(t, env.DB.Postgresql.
		Where("user_id = ? AND organisation_id = ?", invite.InvitedBy, env.OrgID).
		Find(&memberRows).Error)
	assert.Len(t, memberRows, 1, "must not duplicate membership when accepting as existing member")
	assert.Equal(t, beforeMember.RoleID, memberRows[0].RoleID,
		"existing member's role must be preserved across re-acceptance")
}

// TestVerifyEndpoint_AcceptingAfterManualRegistration verifies that accepting
// an invite when the user registered *after* the invitation was sent correctly
// maps to the existing user and does not create a duplicate user row.
func TestVerifyEndpoint_AcceptingAfterManualRegistration(t *testing.T) {
	env := newInviteEnv(t)

	// Create an invitation for a non-existent user.
	// So IsTelexUser is false.
	email := fmt.Sprintf("lateuser%v@qa.team", utility.GenerateUUID())
	tok, _ := utility.GenerateInvitationToken()
	invite := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          tok,
		Status:         "invited",
		Role:           "00000000-0000-0000-0000-000000000000",
		OrganisationID: env.OrgID,
		InvitedBy:      tst.GetUserIDFromToken(t, env.Token, env.DB),
		IsTelexUser:    false,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, env.DB.Postgresql.Create(&invite).Error)

	// Now the user registers manually before accepting the invitation
	uid := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       email,
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "late",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("late_%v", uid),
	}
	authCtrl := auth.Controller{
		Db:        env.DB,
		Validator: validator.New(),
		Logger:    env.InviteCtl.Logger,
		ExtReq:    request.ExternalRequest{Logger: env.InviteCtl.Logger, Test: true},
	}
	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, signUp, false)

	// Verify the user exists in DB exactly once
	var registeredUser models.User
	assert.NoError(t, env.DB.Postgresql.Where("email = ?", email).First(&registeredUser).Error)

	// Accept the invitation via /verify
	body := models.VerifyInvitationLinkRequest{Token: tok}
	rr := env.doJSON(t, http.MethodPost, "/api/v1/invite/verify", body, false)
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	// Assert the invitation is accepted
	var afterInvite models.Invitation
	assert.NoError(t, env.DB.Postgresql.Where("id = ?", invite.ID).First(&afterInvite).Error)
	assert.Equal(t, "accepted", afterInvite.Status)

	// Verify that NO duplicate user record is created in the database
	var users []models.User
	assert.NoError(t, env.DB.Postgresql.Where("email = ?", email).Find(&users).Error)
	assert.Len(t, users, 1, "should not create a duplicate user record")

	// Verify the user was added to the organization
	var memberRows []models.OrgUserManagement
	assert.NoError(t, env.DB.Postgresql.
		Where("user_id = ? AND organisation_id = ?", registeredUser.ID, env.OrgID).
		Find(&memberRows).Error)
	assert.Len(t, memberRows, 1, "user must be added to the organization exactly once")
}

