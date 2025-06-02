package invitation

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
)

type InvitationDetail struct {
	Email string
	Link  string
}

func SendInvitationsEmail(db *gorm.DB, logger *utility.Logger, invitationResponseMap []models.InvitationResponse) error {
	var (
		user models.User
		org  models.Organisation
	)

	err, _ := postgresql.SelectOneFromDb(db, &user, "id = ?", invitationResponseMap[0].InvitedBy)
	if err != nil {
		logger.Error("Failed to get user details", err)
	}

	err, _ = postgresql.SelectOneFromDb(db, &org, "id = ?", invitationResponseMap[0].OrgID)
	if err != nil {
		logger.Error("Failed to get organisation details", err)
	}

	for _, invite := range invitationResponseMap {
		err := SendEmail(invite.Email, invite.InvitationLink, org.Name, user.Name)
		if err != nil {
			logger.Error("Failed to send invitation email", err)
			continue
		}
	}
	return nil
}

func SendEmail(email, link, org_name, inviter_name string) error {
	reqData := models.SendInvitationLink{
		Email:            email,
		InvitationLink:   link,
		OrganisationName: org_name,
		InviterName:      inviter_name,
	}

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendInvitationLink, reqData)
	if err != nil {
		return err
	}
	return nil
}
