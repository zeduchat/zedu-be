package invitation

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
)

func ChannelCheckerValidator(base *storage.Database, inviteReq models.ChannelInvitationCreateReq, owner_id string, logger *utility.Logger) (int, string, error) {
	var (
		o models.Organisation
	)

	org, err := o.CheckOrgExists(inviteReq.OrganisationID, base.Postgresql)
	if err != nil {
		return http.StatusNotFound, "Invalid Organisation ID", err
	}

	isAdmin := CheckUserIsAdmin(base.Postgresql, owner_id, org)
	if !isAdmin {
		return http.StatusUnauthorized, "User is not an admin of the organisation", errors.New("User is not an admin of the organisation")
	}

	if len(inviteReq.Emails) == 0 {
		return http.StatusBadRequest, "No emails provided", errors.New("No emails provided")
	}

	if CheckDuplicateEmails(inviteReq.Emails) {
		return http.StatusBadRequest, "Duplicate emails detected", errors.New("Duplicate emails detected")
	}

	return http.StatusOK, "User validated", nil
}

func ChannelInvitationLinkGenerator(base *storage.Database, inviteReq models.ChannelInvitationCreateReq, userId, url string) ([]models.ChannelInvitation, error) {

	var (
		emails             = inviteReq.Emails
		c                  models.ChannelInvitation
		channelInvitations []models.ChannelInvitation
	)

	for _, email := range emails {
		token, _ := utility.GenerateInvitationToken()

		err := c.ChannelInvitationValidator(base.Postgresql, email, inviteReq)
		if err != nil {
			continue
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

	if len(invitationsMap) == 0 {
		return errors.New("no invitations to save")
	}

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
			InvitationLink: utility.GenerateChannelInvitationLink(baseURL, invite.ChannelID, invite.Token),
			Sent_At:        invite.CreatedAt,
			Expires_At:     invite.ExpiresAt,
		})
	}
	return response
}

func SendChannelsInvitationsEmail(invitationResponseMap []models.ChannelInvitationResponse) error {

	var wg sync.WaitGroup
	errorChannel := make(chan error, len(invitationResponseMap))

	for _, invite := range invitationResponseMap {
		wg.Add(1)
		go func(invite models.ChannelInvitationResponse) {
			defer wg.Done()

			err := sendEmail(invite.Email, invite.InvitationLink)
			if err != nil {
				errorChannel <- fmt.Errorf("failed to send invitation to %s: %v", invite.Email, err)
			}
		}(invite)
	}

	wg.Wait()
	close(errorChannel)

	if len(errorChannel) > 0 {
		var errMsg string
		for err := range errorChannel {
			errMsg += fmt.Sprintf("%v\n", err)
		}
		return fmt.Errorf("some invitations failed to send: \n%s", errMsg)
	}

	return nil
}

func VerifyChannelInvitation(req models.VerifyInvitationLinkRequest, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest) (gin.H, int, error) {

	var (
		user              = models.User{}
		responseData      gin.H
		channelInvitation = models.ChannelInvitation{}
		channel           = models.Channels{}
	)

	exist, err := channelInvitation.GetChannelInvitationLinkByToken(db, req.Token)
	if err != nil {
		return responseData, http.StatusUnauthorized, errors.New("invalid or expired token")
	}

	userData, err := user.GetUserByEmail(db, exist.Email)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("user to be added to channel must be a telex user")
	}

	tokenData, err := middleware.CreateToken(userData, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error saving token")
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	reqs := models.JoinChannelsRequest{
		UserID:     userData.ID,
		ChannelsID: exist.ChannelID,
		Username:   userData.Name,
	}

	_, err = channel.AddUserToChannels(db, reqs)
	if err != nil {
		return responseData, http.StatusInternalServerError, err
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: userData.ID}

	fmt.Printf("Saving access token for user: %s\n", userData.Email)
	err = access_token.CreateAccessToken(db, tokens)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error saving token")
	}

	responseData = gin.H{

		"user": map[string]interface{}{
			"id":           userData.ID,
			"email":        userData.Email,
			"username":     userData.Name,
			"is_onboarded": userData.IsOnboarded,
			"is_verified":  userData.IsVerified,
			"first_name":   userData.Profile.FirstName,
			"last_name":    userData.Profile.LastName,
			"fullname":     userData.Profile.FirstName + " " + userData.Profile.LastName,
			"phone":        userData.Profile.Phone,
			"avatar_url":   userData.Profile.AvatarURL,
			"expires_in":   strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":   strconv.Itoa(int(userData.CreatedAt.Unix())),
			"updated_at":   strconv.Itoa(int(userData.UpdatedAt.Unix())),
		},
		"access_token": tokenData.AccessToken,
	}

	audit_utility.LogUserLogin(c, db, extReq, user.ID, tokenData.AccessUuid, user.Organisations)

	return responseData, http.StatusOK, nil
}
