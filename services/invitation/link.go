package invitation

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/auth"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateInvitation(email, token, role, status string, isTelexUser bool, orgID string) models.Invitation {
	return models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          token,
		Status:         status,
		Role:           role,
		IsTelexUser:    isTelexUser,
		OrganisationID: orgID,
		ExpiresAt:      time.Now().UTC().Add(48 * time.Hour),
	}
}

func InvitationLinkGenerator(base *storage.Database, inviteReq models.InvitationCreateReq, userId, url string) ([]models.Invitation, []string, error) {
	var (
		emails      = inviteReq.Emails
		i           models.Invitation
		invitations []models.Invitation
		errors      []string
	)

	for _, email := range emails {
		token, _ := utility.GenerateInvitationToken()

		creds, err := i.CheckForTelexPresence(base.Postgresql, email, inviteReq.OrganisationID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Error checking for telex presence: %s", err))
		}

		exists := postgresql.CheckExists(base.Postgresql, &models.User{}, "email = ?", email)
		isTelexUser := exists

		//check if the user's email has a pending invitation for that organisation with a pending status
		invitationExists := postgresql.CheckExists(base.Postgresql, &models.Invitation{}, "email = ? AND organisation_id = ? AND status = ?", email, inviteReq.OrganisationID, "invited")
		if invitationExists {
			errors = append(errors, fmt.Sprintf("An invitation has already been sent to %s", email))
			continue
		}

		if creds["RoleID"] != "" || creds["Status"] != "" {
			invitation := CreateInvitation(email, token, creds["RoleID"], creds["Status"], isTelexUser, inviteReq.OrganisationID)
			invitations = append(invitations, invitation)
			continue
		}

		invitation := CreateInvitation(email, token, inviteReq.Role, "invited", isTelexUser, inviteReq.OrganisationID)
		invitations = append(invitations, invitation)
	}
	return invitations, errors, nil
}

func InviteLinkMapper(baseURL string, invitations []models.Invitation) []models.InvitationResponse {
	var response []models.InvitationResponse

	for _, invite := range invitations {
		response = append(response, models.InvitationResponse{
			ID:             invite.ID,
			Email:          invite.Email,
			OrgID:          invite.OrganisationID,
			Status:         "invited",
			InviteToken:    invite.Token,
			IsTelexUser:    invite.IsTelexUser,
			InvitationLink: utility.GenerateInvitationLink(baseURL, invite.OrganisationID, invite.Token),
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

func VerifyInvitation(req models.VerifyInvitationLinkRequest, db *gorm.DB, c *gin.Context) (gin.H, int, error) {

	var (
		user         = models.User{}
		responseData gin.H
		i            = models.Invitation{}
		orgmgt       = models.OrgUserManagement{}
		chans        = models.Channels{}
	)

	invitation, err := i.GetInvitationLinkByToken(db, req.Token)
	if err != nil {
		return responseData, http.StatusUnauthorized, err
	}

	otp, _ := utility.GenerateOTP(6)
	entry := "telex-" + strconv.Itoa(int(otp))

	if invitation.IsTelexUser {
		exists := postgresql.CheckExists(db, &user, "email = ?", invitation.Email)
		if !exists {
			return responseData, http.StatusBadRequest, errors.New("invalid credentials")
		}
		err = i.UpdateInvitation(db, invitation.Email, "accepted")
		if err != nil {
			return responseData, http.StatusInternalServerError, errors.New("error updating invitation")
		}

		orgmgt.RoleID = invitation.Role
		orgmgt.Status = "active"
		orgmgt.UserID = user.ID
		orgmgt.OrganisationID = invitation.OrganisationID

		err = orgmgt.AddUserToOrganisation(db, invitation.OrganisationID, user.ID)
		if err != nil {
			return responseData, http.StatusInternalServerError, err
		}
	} else {

		var user models.User

		arr := strings.Split(invitation.Email, "@")
		email := utility.SplitEmailString(arr[0])

		req := models.CreateUserRequestModel{
			Email:       invitation.Email,
			Password:    entry,
			FirstName:   strings.TrimSpace(strings.ToLower(email)),
			IsOnboarded: true,
		}

		_, _, err := auth.CreateUser(req, db)
		if err != nil {
			return responseData, http.StatusInternalServerError, errors.New("error creating user")
		}

		err = i.UpdateInvitation(db, invitation.Email, "accepted")
		if err != nil {
			return responseData, http.StatusInternalServerError, errors.New("error updating invitation")
		}

		userData, err := user.GetUserByEmail(db, invitation.Email)
		if err != nil {
			return responseData, http.StatusInternalServerError, errors.New("unable to fetch user")
		}

		orgmgt.RoleID = invitation.Role
		orgmgt.Status = "active"
		orgmgt.UserID = userData.ID
		orgmgt.OrganisationID = invitation.OrganisationID

		err = orgmgt.AddUserToOrganisation(db, invitation.OrganisationID, userData.ID)
		if err != nil {
			return responseData, http.StatusInternalServerError, err
		}
	}

	exists := postgresql.CheckExists(db, &chans, "name = ? AND organisation_id = ?", "Default", orgmgt.OrganisationID)
	if !exists {
		return responseData, http.StatusBadRequest, errors.New("channel with name Default and/or channel with organisation ID does not exist")
	}

	exists = postgresql.CheckExists(db, &user, "email = ?", invitation.Email)
	if !exists {
		return responseData, http.StatusBadRequest, errors.New("invalid credentials")
	}

	reqs := models.JoinChannelsRequest{
		Username:   user.Name,
		ChannelsID: chans.ID,
		UserID:     orgmgt.UserID,
	}

	_, err = chans.AddUserToChannels(db, reqs)
	if err != nil {
		return responseData, http.StatusInternalServerError, err
	}

	userData, err := user.GetUserByEmail(db, invitation.Email)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("unable to fetch user")
	}

	tokenData, err := middleware.CreateToken(userData, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error creating token")
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: userData.ID}

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
			"password":     entry,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}
