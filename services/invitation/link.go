package invitation

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func InvitationLinkGenerator(base *storage.Database, inviteReq models.InvitationCreateReq, userId string) ([]models.Invitation, error) {
	//batch create invitations
	var (
		emails      = inviteReq.Emails
		invitations []models.Invitation
	)

	for _, email := range emails {
		token, _ := GenerateInvitationToken()

		//remember: lets check if the email of the user already exists in the organisation so as not to override their roles and status

		invitation := models.Invitation{
			ID:             utility.GenerateUUID(),
			Email:          email,
			Token:          token,
			Status:         "invited",
			Role:           inviteReq.Role,
			OrganisationID: inviteReq.OrganisationID,
			ExpiresAt:      time.Now().Add(24 * time.Hour),
		}

		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

func GenerateInvitationToken() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func InviteLinkMapper(baseURL string, invitations []models.Invitation) []models.InvitationResponse {
	var response []models.InvitationResponse

	for _, invite := range invitations {
		response = append(response, models.InvitationResponse{
			Email:          invite.Email,
			OrgID:          invite.OrganisationID,
			Status:         "invited",
			InviteToken:    invite.Token,
			InvitationLink: GenerateInvitationLink(baseURL, invite.Token),
			Sent_At:        invite.CreatedAt,
			Expires_At:     invite.ExpiresAt,
		})
	}
	return response
}



func ExtractTokenFromInvitationLink(invitationLink string) string {
	splitLink := strings.Split(invitationLink, "/")
	return splitLink[len(splitLink)-1]
}
