package test_invitation

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	svcInvitation "github.com/hngprojects/telex_be/services/invitation"
	"github.com/hngprojects/telex_be/utility"
)

// An existing (lowercased) user invited with a mixed-case address must be
// recognised as a Telex user and stored lowercased.
func TestInvitationLinkGeneratorNormalisesEmailCase(t *testing.T) {
	db, fx := newResendUnitFixture(t)

	var user models.User
	assert.NoError(t, db.Postgresql.Where("id = ?", fx.UserID).First(&user).Error)
	mixed := "  " + strings.ToUpper(user.Email[:1]) + user.Email[1:] + " "

	req := models.InvitationCreateReq{Emails: []string{mixed}, OrganisationID: fx.OrgID, RoleID: fx.RoleID}
	out, errs, err := svcInvitation.InvitationLinkGenerator(db, req, fx.UserID, "http://example.test")
	assert.NoError(t, err)
	assert.Empty(t, errs)
	assert.Len(t, out, 1)
	assert.Equal(t, user.Email, out[0].Email, "invitation email must be stored lowercased and trimmed")
	assert.True(t, out[0].IsTelexUser, "existing user must be detected regardless of case")

	few := models.InvitationCreateFewRequest{
		OrganisationID: fx.OrgID,
		Invitations:    []models.Invite{{Email: mixed, Role: fx.RoleID}},
	}
	outFew, errs, err := svcInvitation.InviteFewLinkGenerator(db, few, fx.UserID, "http://example.test")
	assert.NoError(t, err)
	assert.Empty(t, errs)
	assert.Len(t, outFew, 1)
	assert.Equal(t, user.Email, outFew[0].Email)
	assert.True(t, outFew[0].IsTelexUser)
}

// A pending invitation must be reused when the same address is invited again
// in a different case, for both generators, including legacy mixed-case rows.
func TestInvitationLinkGeneratorReusesPendingRowAnyCase(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	lower := fmt.Sprintf("case%v@example.com", utility.GenerateUUID())
	legacyMixed := "Case" + lower[4:]

	seedID := utility.GenerateUUID()
	tok, _ := utility.GenerateInvitationToken()
	assert.NoError(t, db.Postgresql.Create(&models.Invitation{
		ID: seedID, Email: legacyMixed, Token: tok, Status: "invited", Role: fx.RoleID,
		OrganisationID: fx.OrgID, InvitedBy: fx.UserID, ExpiresAt: time.Now().Add(48 * time.Hour).UTC(),
	}).Error)

	req := models.InvitationCreateReq{Emails: []string{strings.ToUpper(lower)}, OrganisationID: fx.OrgID, RoleID: fx.RoleID}
	out, _, err := svcInvitation.InvitationLinkGenerator(db, req, fx.UserID, "http://example.test")
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, seedID, out[0].ID, "must reuse the pending row instead of creating a second one")

	few := models.InvitationCreateFewRequest{OrganisationID: fx.OrgID, Invitations: []models.Invite{{Email: lower, Role: fx.RoleID}}}
	outFew, _, err := svcInvitation.InviteFewLinkGenerator(db, few, fx.UserID, "http://example.test")
	assert.NoError(t, err)
	assert.Len(t, outFew, 1)
	assert.Equal(t, seedID, outFew[0].ID)
}

// Accepting one invitation must update only that row, not every invitation
// for the same address in the organisation.
func TestUpdateInvitationOnlyTouchesThatRow(t *testing.T) {
	db, fx := newResendUnitFixture(t)
	email := fmt.Sprintf("accept%v@example.com", utility.GenerateUUID())

	mk := func(status string) models.Invitation {
		tok, _ := utility.GenerateInvitationToken()
		inv := models.Invitation{
			ID: utility.GenerateUUID(), Email: email, Token: tok, Status: status, Role: fx.RoleID,
			OrganisationID: fx.OrgID, InvitedBy: fx.UserID, ExpiresAt: time.Now().Add(48 * time.Hour).UTC(),
		}
		assert.NoError(t, db.Postgresql.Create(&inv).Error)
		return inv
	}
	older := mk("accepted")
	pending := mk("invited")

	pending.Status = "accepted"
	assert.NoError(t, pending.UpdateInvitation(db.Postgresql))

	var rows []models.Invitation
	assert.NoError(t, db.Postgresql.Where("email = ? AND organisation_id = ?", email, fx.OrgID).Find(&rows).Error)
	assert.Len(t, rows, 2, "both rows must survive with their own ids")
	for _, r := range rows {
		assert.Equal(t, "accepted", r.Status)
		if r.ID == older.ID {
			assert.Equal(t, older.Token, r.Token, "the other row must not be overwritten")
		}
	}
}
