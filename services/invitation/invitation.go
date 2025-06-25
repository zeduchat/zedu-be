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

func ChangeGeneralInviteStatus(db *gorm.DB, req models.ChangeStatus, logger *utility.Logger, userID string) (int, error) {
	var (
		invite models.GeneralInvitation
	)

	exists := postgresql.CheckExists(db, &invite, "token = ?", req.InvitationID)
	if !exists {
		return http.StatusNotFound, errors.New("invitation does not exists")
	}

	if userID != invite.InvitedBy {
		return http.StatusUnauthorized, errors.New("only invitees can change invitation status")
	}

	err := invite.ChangeGeneralInviteStatus(db, req)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("unable to change general invite status: %s", err)
	}

	return http.StatusOK, nil
}

func GeneralInvitationVerify(db *storage.Database, req models.VerifyShareableInvitationLink, logger *utility.Logger, userID string) (string, int, error) {
	var (
		user   models.User
		invite models.GeneralInvitation
		org    models.Organisation
		orgmgt models.OrgUserManagement
	)

	exists := postgresql.CheckExists(db.Postgresql, &user, "id = ?", userID)
	if !exists {
		return "", http.StatusNotFound, fmt.Errorf("user does not exist")
	}

	err := db.Postgresql.Where("token = ? and active_status = ? AND expires_at > ?",
		req.Token,
		true,
		time.Now().UTC(),
	).First(&invite).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("invitation not found or expired", err)
			return "", http.StatusNotFound, fmt.Errorf("invalid or expired invitation")
		}

		logger.Error("database error", err)
		return "", http.StatusInternalServerError, fmt.Errorf("failed to verify invitation: %s", err)
	}

	_, err = org.CheckOrgExists(invite.OrganisationID, db.Postgresql)
	if err != nil {
		return "", http.StatusNotFound, fmt.Errorf("organisation not found or has been deleted")
	}

	exists = postgresql.CheckExists(db.Postgresql, &orgmgt,
		"user_id = ? AND organisation_id = ?",
		userID, invite.OrganisationID)
	if exists {
		return "", http.StatusConflict, fmt.Errorf("user is already a member of this organisation")
	}

	addToOrg := models.OrgUserManagement{
		UserID:         userID,
		OrganisationID: invite.OrganisationID,
		RoleID:         invite.Role,
		Status:         "active",
	}

	tx := db.Postgresql.Begin()
	if tx.Error != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to start transaction: %s", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("transaction failed", fmt.Errorf("%v", r))
		}
	}()

	code, err := addToOrg.AddUserToOrganisation(tx)
	if err != nil {
		return "", code, fmt.Errorf("unable to add user to organisation: %s", err)
	}

	generalChannel, err := getGeneralChannel(tx, invite.OrganisationID)
	if err != nil {
		logger.Error("error getting default channel", err)
	}

	if generalChannel.ID != "" {
		err = addUserToChannel(&generalChannel, addToOrg, logger, db)
		if err != nil {
			logger.Error("error adding user to the default channel", err)
			return "", http.StatusInternalServerError, err
		}
	}

	if err = tx.Commit().Error; err != nil {
		logger.Error("transaction commit failed", err)
		return "", http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %s", err)
	}

	return "User verified successfully", http.StatusOK, nil
}

func GeneralInvitationCreate(db *gorm.DB, req models.ShareableInviteRequest, user_id, base_url string) (models.ShareableInviteResponse, int, error) {
	var (
		resp   models.ShareableInviteResponse
		invite models.GeneralInvitation
		og     models.Organisation
	)

	_, err := og.CheckOrgExists(req.OrganisationID, db)
	if err != nil {
		return resp, http.StatusNotFound, err
	}

	generateShareableInviteResponse := func(invite models.GeneralInvitation) models.ShareableInviteResponse {
		return models.ShareableInviteResponse{
			InvitationLink: utility.GenerateGeneralInvitationLink(base_url, invite.OrganisationID, invite.Token),
			Expires_At:     invite.ExpiresAt,
			Created_At:     invite.CreatedAt,
		}
	}

	err, _ = postgresql.SelectOneFromDb(db, &invite,
		"organisation_id = ? AND active_status = ? AND expires_at > ?",
		req.OrganisationID,
		true,
		time.Now().UTC(),
	)

	if err == nil {
		resp = generateShareableInviteResponse(invite)
		return resp, http.StatusOK, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := invite.CreateShareableInvite(db, req, user_id); err != nil {
			return resp, http.StatusBadRequest, fmt.Errorf("failed to create invitation: %w", err)
		}
		resp = generateShareableInviteResponse(invite)
		return resp, http.StatusCreated, nil
	}
	return resp, http.StatusInternalServerError, fmt.Errorf("database error: %w", err)
}

func AdminResend(db *gorm.DB, logger *utility.Logger, req models.ResendCondition, baseURL string) (int, error) {
	var (
		invites []models.Invitation
		user    models.User
		org     models.Organisation
	)

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

	err, _ = postgresql.SelectOneFromDb(db, &user, "id = ?", invites[0].InvitedBy)
	if err != nil {
		logger.Error("Failed to get user details", err)
	}

	err, _ = postgresql.SelectOneFromDb(db, &org, "id = ?", invites[0].OrganisationID)
	if err != nil {
		logger.Error("Failed to get organisation details", err)
	}

	for _, invite := range invites {
		invitation_link := utility.GenerateInvitationLink(baseURL, invite.OrganisationID, invite.Token)

		err := SendEmail(invite.Email, invitation_link, org.Name, user.Name)
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

func CheckerValidator(base *storage.Database, Emails []string, ids models.IDS, logger *utility.Logger) (int, string, error) {
	var (
		o models.Organisation
		r models.OrgRole
	)
	_, err := o.CheckOrgExists(ids.OrganisationID, base.Postgresql)
	if err != nil {
		return http.StatusNotFound, "Invalid Organisation ID", err
	}

	if len(Emails) == 0 {
		return http.StatusBadRequest, "No emails provided", errors.New("no emails provided")
	}

	if CheckDuplicateEmails(Emails, []models.Invite{}, false) {
		return http.StatusBadRequest, "Duplicate emails detected", errors.New("duplicate emails detected")
	}

	exists := postgresql.CheckExists(base.Postgresql, &r, "id = ?", ids.RoleID)
	if !exists {
		return http.StatusNotFound, "Role does not exist", errors.New("role does not exist")
	}

	return http.StatusOK, "User(s) validated", nil
}

func FewInvitesCheckerValidator(base *storage.Database, ids models.IDS, invitations []models.Invite) (int, string, error) {
	var (
		o models.Organisation
	)
	_, err := o.CheckOrgExists(ids.OrganisationID, base.Postgresql)
	if err != nil {
		return http.StatusNotFound, "Invalid Organisation ID", err
	}

	if len(invitations) == 0 {
		return http.StatusBadRequest, "No invitations provided", errors.New("no invitations provided")
	}

	if CheckDuplicateEmails([]string{}, invitations, true) {
		return http.StatusBadRequest, "Duplicate emails detected", errors.New("duplicate emails detected")
	}

	err = ValidateRoles(base.Postgresql, invitations)
	if err != nil {
		return http.StatusBadRequest, "invalid role assigned", errors.New("invalid role assigned")
	}

	return http.StatusOK, "User(s) validated", nil
}

func ValidateRoles(db *gorm.DB, invitations []models.Invite) error {
	var role models.OrgRole

	for _, invite := range invitations {
		exists := postgresql.CheckExists(db, role, "id = ?", invite.Role)
		if !exists {
			return fmt.Errorf("invalid role-id assigned to user %s", invite.Email)
		}
	}

	return nil
}

func CheckUserIsAdmin(db *gorm.DB, owner_id string, org models.Organisation) bool {
	return org.OwnerID == owner_id
}

func CheckDuplicateEmails(emails []string, invitations []models.Invite, checkinvitefew bool) bool {
	if checkinvitefew {
		emailsMap := make(map[string]struct{})
		for _, invite := range invitations {
			if _, exist := emailsMap[invite.Email]; exist {
				return true
			}
			emailsMap[invite.Email] = struct{}{}
		}
		return false
	}

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
		return invitation, errors.New("invitation link does not exist")
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
