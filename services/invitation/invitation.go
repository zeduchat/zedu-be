package invitation

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func AdminInvitationCreate(db *gorm.DB, req models.ShareableInviteRequest, user_id, base_url string) (models.ShareableInviteResponse, int, error) {
	var (
		resp   models.ShareableInviteResponse
		invite models.GeneralInvitation
		og     models.Organisation
	)

	org, err := og.CheckOrgExists(req.OrganisationID, db)
	if err != nil {
		return resp, http.StatusNotFound, err
	}

	if org.OwnerID != user_id {
		return resp, http.StatusUnauthorized, fmt.Errorf("only organisation admins can create invitation")
	}

	err, _ = postgresql.SelectOneFromDb(db, &invite, "organisation_id = ? AND status=? AND expires_at > ?",
		req.OrganisationID,
		"active",
		time.Now().UTC())

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := invite.CreateShareableInvite(db, req, user_id); err != nil {
				return resp, http.StatusBadRequest, err
			}

			token := 

			resp.InvitationLink = utility.GenerateInvitationLink(base_url, invite.OrganisationID, invite.Token)
			resp.Created_At = invite.CreatedAt
			resp.Expires_At = invite.ExpiresAt
			return resp, http.StatusCreated, nil

		}
	}
	
}


func AdminResend(db *gorm.DB, logger *utility.Logger, req models.ResendCondition, baseURL string) (int, error) {
	var invites []models.Invitation

	//parse the date
	parsed_date, err := time.Parse("2006-01-02", req.TimeFrom)
	if err != nil {
		return 400, fmt.Errorf("error parsing the time into the right format")
	}

	err = db.Where("email LIKE ? AND status = 'invited'  AND created_at BETWEEN ? AND ?",
		"%"+req.Extension,
		parsed_date,
		time.Now().UTC(),
	).Find(&invites).Error

	if err != nil {
		return 500, fmt.Errorf("failed to fetch users: %w", err)
	}

	successful_reinvites := []string{}

	for _, invite := range invites {
		invitation_link := utility.GenerateInvitationLink(baseURL, invite.OrganisationID, invite.Token)

		err := SendEmail(invite.Email, invitation_link)
		if err != nil {
			continue
		}

		successful_reinvites = append(successful_reinvites, invite.Email)
	}

	if len(successful_reinvites) > 0 {
		err := db.Model(&models.Invitation{}).
			Where("email IN ?", successful_reinvites).
			Updates(map[string]any{
				"expires_at": time.Now().Add(48 * time.Hour),
			})

		if err != nil {
			logger.Error("error encountered updating reinvited emails expiration time")
		}
	}

	return http.StatusOK, nil
}

func CheckerValidator(base *storage.Database, Emails []string, OrganisationID string, userId string, logger *utility.Logger) (int, string, error) {
	var o models.Organisation

	_, err := o.CheckOrgExists(OrganisationID, base.Postgresql)
	if err != nil {
		return http.StatusNotFound, "Invalid Organisation ID", err
	}

	if len(Emails) == 0 {
		return http.StatusBadRequest, "No emails provided", errors.New("no emails provided")
	}

	if CheckDuplicateEmails(Emails) {
		return http.StatusBadRequest, "Duplicate emails detected", errors.New("duplicate emails detected")
	}

	return http.StatusOK, "User validated", nil
}

func CheckUserIsAdmin(db *gorm.DB, owner_id string, org models.Organisation) bool {
	return org.OwnerID == owner_id
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

	exists := postgresql.CheckExists(db, &invitation, "token = ?", token)
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
		return invitation, "Invitation link expired", errors.New("invitation link expired")
	}

	if invitation.Status == "accepted" {
		return invitation, "Invitation link already accepted", errors.New("invitation link already accepted")
	}

	if invitation.OrganisationID == "" {
		return invitation, "Invalid organisation ID", errors.New("invalid organisation ID")
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

func ResendLinkGenerator(base *storage.Database, logger *utility.Logger, req models.ResendInvitationRequest) (models.Invitation, error) {

	var (
		email = req.Email
		i     models.Invitation
	)

	invite, pending, _ := i.CheckPendingInvitations(base.Postgresql, email, req.OrganisationID)

	if !pending {
		return invite, fmt.Errorf("user with email %s has already accepted invitation", email)
	}

	invite.Token, _ = utility.GenerateInvitationToken()
	invite.ExpiresAt = time.Now().Add(48 * time.Hour)
	invite.CreatedAt = time.Now()

	err := invite.UpdateResendInvitation(base.Postgresql, email)
	if err != nil {
		return models.Invitation{}, fmt.Errorf("failed to update invitation for %s", email)
	}

	return invite, nil
}

func CancelInvitation(db *gorm.DB, inviteID, userID string) error {
	var (
		i models.Invitation
	)

	return i.DeleteInvitation(db, inviteID)
}
