package invitation

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
)

type InvitationDetail struct {
	Email string
	Link  string
}

// use for loops with a sleep perid of 0.5 seconds to send emails concurrently.. do not use goroutine
func SendInvitationsEmail(invitationResponseMap []models.InvitationResponse) error {
	for _, invite := range invitationResponseMap {
		err := sendEmail(invite.Email, invite.InvitationLink)
		if err != nil {
			return fmt.Errorf("failed to send invitation to %s: %v", invite.Email, err)
		}
	}
	return nil

}

func sendEmail(email, link string) error {
	reqData := models.SendInvitationLink{
		Email:          email,
		InvitationLink: link,
	}

	send := fmt.Sprintf("Sending invitation email to %s with link %s ", email, link)
	fmt.Println(send)

	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendInvitationLink, reqData)
	if err != nil {
		return err
	}

	return nil
}
