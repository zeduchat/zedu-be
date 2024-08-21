package invitation

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CheckerValidator(base *storage.Database, inviteReq models.InvitationCreateReq, userId string, logger *utility.Logger) (int, string, error) {
	var o models.Organisation


	org, err := o.CheckOrgExists(inviteReq.OrganisationID, base.Postgresql)
	if err != nil {
		return http.StatusNotFound, "Invalid Organisation ID", err
	}

	isAdmin := CheckUserIsAdmin(base.Postgresql, userId, org)
	if !isAdmin {
		return http.StatusUnauthorized, "User is not an admin of the organisation", errors.New("User is not an admin of the organisation")
	}

	if len(inviteReq.Emails) == 0 {
		return http.StatusBadRequest, "No emails provided", errors.New("No emails provided")
	}

	if CheckEmailsLimit(inviteReq.Emails) {
		return http.StatusBadRequest, "Emails limit exceeded", errors.New("Emails limit exceeded")
	}

	if CheckDuplicateEmails(inviteReq.Emails) {
		return http.StatusBadRequest, "Duplicate emails detected", errors.New("Duplicate emails detected")
	}

	return http.StatusOK, "User validated", nil
}

func CheckUserIsAdmin(db *gorm.DB, owner_id string, org models.Organisation) bool {
	return org.OwnerID == owner_id
}

func CheckEmailsLimit(emails []string) bool {
	return len(emails) > 50
}

func CheckDuplicateEmails(emails []string) bool {
	emailsMap := make(map[string]bool)
	for _, email := range emails {
		if _, ok := emailsMap[email]; ok {
			return true
		}
		emailsMap[email] = true
	}
	return false
}

func GenerateInvitationLink(baseurl, orgID, token string) string {
	return baseurl + fmt.Sprintf("/accept_org_invitation?org_id=%s&invitation_token=%s", orgID, token)
}
func GenerateChannelInvitationLink(baseurl, channelID, token string) string {
	return baseurl + fmt.Sprintf("/accept_channel_invitation?channel_id=%s&invitation_token=%s", channelID, token)
}

func SaveInvitations(db *gorm.DB, invitationsMap []models.Invitation) error {
	var (
		i models.Invitation
	)

	err := i.CreateInvitations(db, invitationsMap)
	if err != nil {
		return err
	}
	return nil
}

func GetInvitationDetails(token string, db *gorm.DB) (models.Invitation, error) {
	var invitation models.Invitation
	// Check if the invitation token exists in the database
	exists := postgresql.CheckExists(db, &invitation, "token = ?", token)
	// If it does, return the invitation details
	if !exists {
		return invitation, errors.New("Invitation link does not exist")
	}
	return invitation, nil
}

func AcceptInvitationLink(user_id string, token string, db *gorm.DB) (models.Invitation, string, error) {

	invitation, err := GetInvitationDetails(token, db)
	if err != nil {
		return invitation, "Error getting invitation details", err
	}
	if invitation.ExpiresAt.Before(time.Now()) {
		return invitation, "Invitation link expired", errors.New("Invitation link expired")
	}

	if invitation.Status == "accepted" {
		return invitation, "Invitation link already accepted", errors.New("Invitation link already accepted")
	}

	if invitation.OrganisationID == "" {
		return invitation, "Invalid organisation ID", errors.New("Invalid organisation ID")
	}

	_, err = invitation.ProcessInvitationAcceptance(db, user_id)
	if err != nil {
		return invitation, "Failed to process invitation acceptance", err
	}

	return invitation, "Invitation link accepted successfully", nil
}

func AddUserToOrganisation(db *gorm.DB, orgID string, userId string) error {
	var user models.User

	user, err := user.GetUserByID(db, userId)
	if err != nil {
		return err
	}

	var org models.Organisation
	org, err = org.GetOrgByID(db, orgID)
	if err != nil {
		return err
	}

	err = user.AddUserToOrganisation(db, &user, []interface{}{&org})
	if err != nil {
		return err
	}
	return nil
}

