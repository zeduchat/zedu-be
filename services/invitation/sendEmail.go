package invitation

import (
	"fmt"
	"sync"

	"github.com/hngprojects/telex_be/internal/models"
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

// Mock function to simulate sending an email
func sendEmail(email, link string) error {
	fmt.Printf("Sending invitation to %s with link: %s\n", email, link)
	return nil
}
