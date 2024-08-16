package invitation

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func ChannelCheckerValidator(base *storage.Database, inviteReq models.ChannelInvitationCreateReq, userId string, logger *utility.Logger) (int, string, error) {
	var (
		o models.Organisation
		c models.Channels
	)

	org, err := o.CheckOrgExists(inviteReq.OrganisationID, base.Postgresql)
	if err != nil {
		return http.StatusNotFound, "Invalid Organisation ID", err
	}

	exists := c.CheckChannelExistsInOrg(base.Postgresql, inviteReq.ChannelID, inviteReq.OrganisationID)
	if !exists {
		return http.StatusNotFound, "Channel does not exist in the organisation", errors.New("Channel does not exist in the organisation")
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

func ChannelInvitationLinkGenerator(base *storage.Database, inviteReq models.ChannelInvitationCreateReq, userId, url string) ([]models.ChannelInvitation, error) {
	//batch create invitations
	var (
		emails             = inviteReq.Emails
		c                  models.ChannelInvitation
		channelInvitations []models.ChannelInvitation
	)

	for _, email := range emails {
		token, _ := GenerateInvitationToken()

		//remember: lets check if the email of the user already exists in the organisation so as not to override their roles and status
		err := c.CheckForChannelPresence(base.Postgresql, email, inviteReq.OrganisationID)
		if err != nil {
			fmt.Println("Error checking for channel presence", err)
		}

		invitation := CreateChannelInvitation(email, token, "invited", inviteReq)
		channelInvitations = append(channelInvitations, invitation)
	}
	return channelInvitations, nil
}

func CreateChannelInvitation(email, token, status string, inviteReq models.ChannelInvitationCreateReq) models.ChannelInvitation {
	return models.ChannelInvitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          token,
		Status:         status,
		ChannelID:      inviteReq.ChannelID,
		OrganisationID: inviteReq.OrganisationID,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}
}

func SaveChannelInvitations(db *gorm.DB, invitationsMap []models.ChannelInvitation) error {
	var (
		c models.ChannelInvitation
	)

	err := c.CreateChannelInvitations(db, invitationsMap)
	if err != nil {
		return err
	}
	return nil
}

func ChannelInviteLinkMapper(baseURL string, invitations []models.ChannelInvitation) []models.ChannelInvitationResponse {
	var response []models.ChannelInvitationResponse

	for _, invite := range invitations {
		response = append(response, models.ChannelInvitationResponse{
			Email:          invite.Email,
			OrgID:          invite.OrganisationID,
			Status:         "invited",
			ChannelID:      invite.ChannelID,
			InviteToken:    invite.Token,
			InvitationLink: GenerateInvitationLink(baseURL, invite.OrganisationID, invite.Token),
			Sent_At:        invite.CreatedAt,
			Expires_At:     invite.ExpiresAt,
		})
	}
	return response
}

func SendChannelsInvitationsEmail(invitationResponseMap []models.ChannelInvitationResponse) error {

	var wg sync.WaitGroup
	errorChannel := make(chan error, len(invitationResponseMap))

	// Iterate through the map and send invitations concurrently
	for _, invite := range invitationResponseMap {
		wg.Add(1)
		go func(invite models.ChannelInvitationResponse) {
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

// func sendEmail(email, link string) error {
// 	reqData := models.SendInvitationLink{
// 		Email:          email,
// 		InvitationLink: link,
// 	}

// 	send := fmt.Sprintf("Sending invitation email to %s with link %s ", email, link)
// 	fmt.Println(send)

// 	err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendInvitationLink, reqData)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
