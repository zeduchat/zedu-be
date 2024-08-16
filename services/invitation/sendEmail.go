package invitation

import (
	"fmt"
	"sync"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
)

type InvitationDetail struct {
	Email string
	Link  string
}

func SendInvitationsEmail(invitationResponseMap []models.InvitationResponse) error {

	var wg sync.WaitGroup
	errorChannel := make(chan error, len(invitationResponseMap))

	// Iterate through the map and send invitations concurrently
	for _, invite := range invitationResponseMap {
		wg.Add(1)
		go func(invite models.InvitationResponse) {
			defer wg.Done()

			// Simulate sending an email
			err := sendEmail(invite.Email, invite.InvitationLink)
			if err != nil {
				errorChannel <- fmt.Errorf("failed to send invitation to %s: %v", invite.Email, err)
			}
		}(invite)
	}

	// Wait for all Goroutines to finish
	wg.Wait()
	close(errorChannel)

	// Check for errors
	if len(errorChannel) > 0 {
		var errMsg string
		for err := range errorChannel {
			errMsg += fmt.Sprintf("%v\n", err)
		}
		return fmt.Errorf("some invitations failed to send: \n%s", errMsg)
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
