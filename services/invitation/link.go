package invitation

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
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
		invitations []models.Invitation
		errs        []string
	)

	for _, email := range emails {
		isTelexUser := postgresql.CheckExists(base.Postgresql, &models.User{}, "email = ?", email)

		// Check if the user's email has a pending invitation for that organisation with a pending status
		invitationExists := postgresql.CheckExists(base.Postgresql, &models.Invitation{}, "email = ? AND organisation_id = ? AND status = 'invited' AND expires_at > ?", email, inviteReq.OrganisationID, time.Now().UTC())
		if invitationExists {
			errs = append(errs, fmt.Sprintf("%s already has a pending invitation.", email))
			continue
		}

		if isTelexUser {
			alreadyMember := postgresql.CheckExists(base.Postgresql, &models.OrgUserManagement{}, "user_id = ? AND organisation_id = ?", userId, inviteReq.OrganisationID)
			if alreadyMember {
				errs = append(errs, fmt.Errorf("%s is already a member of the organisation", email).Error())
				continue
			}
		}

		token, err := utility.GenerateInvitationToken()
		if err != nil {
			errs = append(errs, fmt.Sprintf("could not generate token for %s: %v", email, err))
			continue
		}

		invitation := CreateInvitation(email, token, inviteReq.RoleID, "invited", isTelexUser, inviteReq.OrganisationID)
		invitations = append(invitations, invitation)
	}

	return invitations, errs, nil
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

func VerifyInvitation(req models.VerifyInvitationLinkRequest, db *gorm.DB, c *gin.Context, extReq request.ExternalRequest) (gin.H, int, error) {

	var (
		user         = models.User{}
		responseData gin.H
		i            = models.Invitation{}
		orgmgt       = models.OrgUserManagement{}
		chans        = models.Channels{}
		userID       string
	)

	invitation, code, err := i.GetInvitationLinkByToken(db, req.Token)
	if err != nil {
		return responseData, code, err
	}

	user, err = getOrCreateUser(invitation, db)
	if err != nil {
		return responseData, http.StatusInternalServerError, err
	}

	userID = user.ID
	invitation.Status = "accepted"

	err = invitation.UpdateInvitation(db)
	if err != nil {
		return responseData, http.StatusBadRequest, err
	}

	orgmgt.RoleID = invitation.Role
	orgmgt.Status = "active"
	orgmgt.UserID = userID
	orgmgt.OrganisationID = invitation.OrganisationID

	err = addUserToOrganisation(orgmgt, db)
	if err != nil {
		return responseData, http.StatusInternalServerError, err
	}

	if err = verifyDefaultChannel(&chans, invitation.OrganisationID, db); err != nil {
		return responseData, http.StatusInternalServerError, err
	}

	err = addUserToChannel(&chans, orgmgt, user.Name, db)
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

	err = saveAccessToken(tokenData, userID, db)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error saving access token")
	}

	responseData = buildUserResponse(userData, tokenData)

	return responseData, http.StatusOK, nil
}

func getOrCreateUser(invitation models.Invitation, db *gorm.DB) (models.User, error) {
	var user models.User

	if invitation.IsTelexUser {
		exists := postgresql.CheckExists(db, &user, "email = ?", invitation.Email)
		if !exists {
			return user, errors.New("invalid credentials. User does not exist")
		}

	} else {

		arr := strings.Split(invitation.Email, "@")
		email := utility.SplitEmailString(arr[0])
		name := strings.TrimSpace(strings.ToLower(email))
		orgId, _ := uuid.FromString(invitation.OrganisationID)

		user = models.User{
			ID:          utility.GenerateUUID(),
			Name:        name,
			Email:       invitation.Email,
			IsOnboarded: true,
			IsActive:    true,
			CurrentOrg:  orgId,
			Profile: models.Profile{
				ID: utility.GenerateUUID(),
			},
		}

		err := user.CreateUser(db)
		if err != nil {
			return user, err
		}

		fmt.Println(user.Email)

		resetReq := models.SendWelcomeMail{
			Email: user.Email,
		}

		err = actions.AddNotificationToQueue(storage.DB.Redis, names.SendWelcomeMail, resetReq)
		if err != nil {
			return user, err
		}

		user, err = user.GetUserByEmail(db, invitation.Email)
		if err != nil {
			return user, errors.New("unable to fetch user")
		}

	}

	return user, nil
}

func addUserToOrganisation(orgmgt models.OrgUserManagement, db *gorm.DB) error {

	if err := orgmgt.AddUserToOrganisation(db); err != nil {
		return err
	}

	return nil
}

func verifyDefaultChannel(chans *models.Channels, organisationID string, db *gorm.DB) error {
	exists := postgresql.CheckExists(db, &chans, "name = ? AND organisation_id = ?", "Default", organisationID)
	if !exists {
		return errors.New("channel with name Default and/or channel with organisation ID does not exist")
	}

	return nil
}

func addUserToChannel(chans *models.Channels, orgmgt models.OrgUserManagement, username string, db *gorm.DB) error {
	reqs := models.JoinChannelsRequest{
		Username:   username,
		ChannelsID: chans.ID,
		UserID:     orgmgt.UserID,
	}

	if _, err := chans.AddUserToChannels(db, reqs); err != nil {
		return err
	}
	return nil
}

func saveAccessToken(tokenData *middleware.TokenDetailDTO, userID string, db *gorm.DB) error {

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: userID}
	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	if err := access_token.CreateAccessToken(db, tokens); err != nil {
		return err
	}
	return nil
}

func buildUserResponse(user models.User, tokenData *middleware.TokenDetailDTO) gin.H {
	return gin.H{
		"user": map[string]interface{}{
			"id":              user.ID,
			"email":           user.Email,
			"username":        user.Name,
			"is_onboarded":    user.IsOnboarded,
			"is_verified":     user.IsVerified,
			"profile_updated": user.ProfileUpdated,
			"first_name":      user.Profile.FirstName,
			"last_name":       user.Profile.LastName,
			"fullname":        user.Profile.FirstName + " " + user.Profile.LastName,
			"phone":           user.Profile.Phone,
			"avatar_url":      user.Profile.AvatarURL,
			"current_org":     user.CurrentOrg,
			"expires_in":      strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
			"created_at":      strconv.Itoa(int(user.CreatedAt.Unix())),
			"updated_at":      strconv.Itoa(int(user.UpdatedAt.Unix())),
		},
		"access_token": tokenData.AccessToken,
	}
}
