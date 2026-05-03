package test_invitation

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	svcInvitation "github.com/hngprojects/telex_be/services/invitation"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// resendUnitFixture provisions a real user, organisation and default role in
// the test DB and returns the IDs needed by the unit tests below.
type resendUnitFixture struct {
	UserID string
	OrgID  string
	RoleID string
}

func newResendUnitFixture(t *testing.T) (*storage.Database, *resendUnitFixture) {
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
	userID := tst.GetUserIDFromToken(t, token, db)

	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger}
	createOrg := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("org%v", uid),
		Description: "unit fixture org",
		Email:       signUp.Email,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrg, token)

	role := models.OrgRole{}
	defaultRole, err := role.GetAOrgRoleByName(db.Postgresql, models.OrgRoleNameUser)
	if err != nil {
		t.Fatalf("could not load default org role: %v", err)
	}

	return db, &resendUnitFixture{
		UserID: userID,
		OrgID:  orgID,
		RoleID: defaultRole.ID,
	}
}

// TestCreateInvitationsRotatesTokenOnExisting verifies Fix 2: when a pending
// invitation already exists, CreateInvitations must persist the new token,
// new expiry, and refreshed created_at — not just bump expires_at.
func TestCreateInvitationsRotatesTokenOnExisting(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	email := fmt.Sprintf("rotate%v@example.com", utility.GenerateUUID())

	originalToken, err := utility.GenerateInvitationToken()
	assert.NoError(t, err)

	original := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          originalToken,
		Status:         "invited",
		Role:           fx.RoleID,
		OrganisationID: fx.OrgID,
		InvitedBy:      fx.UserID,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	i := models.Invitation{}
	assert.NoError(t, i.CreateInvitations(db.Postgresql, []models.Invitation{original}))

	var firstSnapshot models.Invitation
	exists := postgresql.CheckExists(db.Postgresql, &firstSnapshot, "email = ? AND organisation_id = ?", email, fx.OrgID)
	assert.True(t, exists, "first invite should be persisted")
	assert.Equal(t, originalToken, firstSnapshot.Token)

	// Sleep a moment so created_at delta is observable.
	time.Sleep(1100 * time.Millisecond)

	rotatedToken, err := utility.GenerateInvitationToken()
	assert.NoError(t, err)
	assert.NotEqual(t, originalToken, rotatedToken)

	resend := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          rotatedToken,
		Status:         "invited",
		Role:           fx.RoleID,
		OrganisationID: fx.OrgID,
		InvitedBy:      fx.UserID,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, i.CreateInvitations(db.Postgresql, []models.Invitation{resend}))

	var afterResend models.Invitation
	exists = postgresql.CheckExists(db.Postgresql, &afterResend, "email = ? AND organisation_id = ?", email, fx.OrgID)
	assert.True(t, exists)

	// Token should have rotated.
	assert.Equal(t, rotatedToken, afterResend.Token,
		"CreateInvitations must persist the new token on a duplicate-email update")

	// Created_at should have moved forward (resend is treated as a fresh send).
	assert.True(t, afterResend.CreatedAt.After(firstSnapshot.CreatedAt),
		"created_at should advance on resend; got first=%v after=%v",
		firstSnapshot.CreatedAt, afterResend.CreatedAt)

	// Should still be only one row for this email/org pair.
	var rows []models.Invitation
	assert.NoError(t, db.Postgresql.Where("email = ? AND organisation_id = ?", email, fx.OrgID).Find(&rows).Error)
	assert.Len(t, rows, 1, "duplicate invite must update in place, not insert a new row")
}

// TestInvitationLinkGeneratorReusesPendingRow verifies Fix 1: when a pending
// invite exists for the email+org, the link generator must (a) NOT raise an
// error, (b) reuse the existing record's ID so we update in place, and (c)
// rotate the token in the returned slice.
func TestInvitationLinkGeneratorReusesPendingRow(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	email := fmt.Sprintf("pending%v@example.com", utility.GenerateUUID())

	// Seed a pending invite directly in the DB.
	originalToken, _ := utility.GenerateInvitationToken()
	originalID := utility.GenerateUUID()
	seed := models.Invitation{
		ID:             originalID,
		Email:          email,
		Token:          originalToken,
		Status:         "invited",
		Role:           fx.RoleID,
		OrganisationID: fx.OrgID,
		InvitedBy:      fx.UserID,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, db.Postgresql.Create(&seed).Error)

	req := models.InvitationCreateReq{
		Emails:         []string{email},
		OrganisationID: fx.OrgID,
		RoleID:         fx.RoleID,
	}

	out, errs, err := svcInvitation.InvitationLinkGenerator(db, req, fx.UserID, "http://example.test")
	assert.NoError(t, err)
	assert.Empty(t, errs, "re-inviting a pending email must not produce per-email errors")
	assert.Len(t, out, 1)

	got := out[0]
	assert.Equal(t, originalID, got.ID, "must reuse the existing row's ID, not invent a new one")
	assert.NotEqual(t, originalToken, got.Token, "token should be rotated for the resend")
	assert.True(t, got.ExpiresAt.After(time.Now().Add(40*time.Hour)), "expiry should be refreshed forward")
}

// TestInvitationLinkGeneratorAllowsMember verifies that re-inviting an email
// that already belongs to a member of the org no longer produces an error —
// the invite should proceed and a record should be returned. The accept flow
// (covered by TestVerifyInvitation_AlreadyMemberSucceeds) is responsible for
// resolving the membership cleanly when the recipient clicks the link.
func TestInvitationLinkGeneratorAllowsMember(t *testing.T) {
	db, fx := newResendUnitFixture(t)

	// The inviter is already a member of their own org; previously this
	// would short-circuit with an "already a member" error.
	var inviter models.User
	assert.NoError(t, db.Postgresql.Where("id = ?", fx.UserID).First(&inviter).Error)

	req := models.InvitationCreateReq{
		Emails:         []string{inviter.Email},
		OrganisationID: fx.OrgID,
		RoleID:         fx.RoleID,
	}

	out, errs, err := svcInvitation.InvitationLinkGenerator(db, req, fx.UserID, "http://example.test")
	assert.NoError(t, err)
	assert.Empty(t, errs, "re-inviting a member must not produce per-email errors")
	assert.Len(t, out, 1, "an invitation record should still be produced for an existing member")
	assert.Equal(t, inviter.Email, out[0].Email)
	assert.NotEmpty(t, out[0].Token)
}

// TestResendLinkGeneratorNoInvitation verifies Fix 4: when there is no
// invitation for the email at all, the resend service should say so — not
// claim the user "already accepted" the invitation.
func TestResendLinkGeneratorNoInvitation(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	logger := tst.Setup()

	req := models.ResendInvitationRequest{
		Email:          fmt.Sprintf("never-invited%v@example.com", utility.GenerateUUID()),
		OrganisationID: fx.OrgID,
	}

	_, err := svcInvitation.ResendLinkGenerator(db, logger, req)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no invitation found"),
		"expected 'no invitation found' message, got %q", err.Error())
}

// TestResendLinkGeneratorAccepted verifies Fix 4: when the invitation exists
// but is already accepted, the resend service correctly reports that.
func TestResendLinkGeneratorAccepted(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	logger := tst.Setup()

	email := fmt.Sprintf("accepted%v@example.com", utility.GenerateUUID())
	tok, _ := utility.GenerateInvitationToken()
	accepted := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          tok,
		Status:         "accepted",
		Role:           fx.RoleID,
		OrganisationID: fx.OrgID,
		InvitedBy:      fx.UserID,
		ExpiresAt:      time.Now().Add(48 * time.Hour).UTC(),
	}
	assert.NoError(t, db.Postgresql.Create(&accepted).Error)

	req := models.ResendInvitationRequest{Email: email, OrganisationID: fx.OrgID}
	_, err := svcInvitation.ResendLinkGenerator(db, logger, req)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "already accepted"),
		"expected 'already accepted' message, got %q", err.Error())
}

// TestResendLinkGeneratorRotatesPending verifies Fix 4 happy path: when a
// pending invitation exists, resend rotates the token and refreshes expiry.
func TestResendLinkGeneratorRotatesPending(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	logger := tst.Setup()

	email := fmt.Sprintf("pending-resend%v@example.com", utility.GenerateUUID())
	originalToken, _ := utility.GenerateInvitationToken()
	pending := models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          originalToken,
		Status:         "invited",
		Role:           fx.RoleID,
		OrganisationID: fx.OrgID,
		InvitedBy:      fx.UserID,
		ExpiresAt:      time.Now().Add(time.Hour).UTC(),
	}
	assert.NoError(t, db.Postgresql.Create(&pending).Error)

	req := models.ResendInvitationRequest{Email: email, OrganisationID: fx.OrgID}
	out, err := svcInvitation.ResendLinkGenerator(db, logger, req)
	assert.NoError(t, err)
	assert.NotEqual(t, originalToken, out.Token, "resend must produce a new token")

	var stored models.Invitation
	assert.True(t, postgresql.CheckExists(db.Postgresql, &stored, "email = ? AND organisation_id = ?", email, fx.OrgID))
	assert.Equal(t, out.Token, stored.Token, "DB row must reflect the rotated token")
	assert.True(t, stored.ExpiresAt.After(time.Now().Add(40*time.Hour)),
		"expiry should be pushed out by ~48h on resend")
}
