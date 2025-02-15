package invitation

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
)

type InvitationDetail struct {
	Email string
	Link  string
}

func SendInvitationsEmail(logger *utility.Logger, invitationResponseMap []models.InvitationResponse) error {
	for _, invite := range invitationResponseMap {
		err := SendEmail(invite.Email, invite.InvitationLink)
		if err != nil {
			logger.Error("Failed to send invitation email", err)
			continue
		}
	}
	return nil
}

func SendEmail(email, link string) error {
	reqData := models.SendInvitationLink{
		Email:          email,
		InvitationLink: link,
	}

	fmt.Println("added to queue", email, link)

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendInvitationLink, reqData)
	if err != nil {
		return err
	}
	return nil
}
