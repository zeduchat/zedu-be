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

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/utility"
)

func CreateInvitation(email, token, status string, isTelexUser bool, ids models.IDS) models.Invitation {
	return models.Invitation{
		ID:             utility.GenerateUUID(),
		Email:          email,
		Token:          token,
		Status:         status,
		Role:           ids.RoleID,
		IsTelexUser:    isTelexUser,
		InvitedBy:      ids.UserID,
		OrganisationID: ids.OrganisationID,
		ExpiresAt:      time.Now().UTC().Add(48 * time.Hour),
	}
}

func InvitationLinkGenerator(base *storage.Database, inviteReq models.InvitationCreateReq, userId, url string) ([]models.Invitation, []string, error) {
	var (
		emails      = inviteReq.Emails
		invitations []models.Invitation
		errs        []string
		user        models.User
	)

	for _, email := range emails {
		var existingInvite models.Invitation
		isTelexUser := postgresql.CheckExists(base.Postgresql, &user, "email = ?", email)

		// Check if the user's email has a pending invitation for that organisation with a pending status
		invitationExists := postgresql.CheckExists(base.Postgresql, &existingInvite, "email = ? AND organisation_id = ? AND status = 'invited' AND expires_at > ?", email, inviteReq.OrganisationID, time.Now().UTC())
		if invitationExists {
			// errs = append(errs, fmt.Sprintf("%s already has a pending invitation.", email))
			invitations = append(invitations, existingInvite)
			continue
		}

		if isTelexUser {
			alreadyMember := postgresql.CheckExists(base.Postgresql, &models.OrgUserManagement{}, "user_id = ? AND organisation_id = ?", user.ID, inviteReq.OrganisationID)
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

		ids := models.IDS{
			UserID:         userId,
			OrganisationID: inviteReq.OrganisationID,
			RoleID:         inviteReq.RoleID,
		}

		invitation := CreateInvitation(email, token, "invited", isTelexUser, ids)
		invitations = append(invitations, invitation)
	}

	return invitations, errs, nil
}

func InviteFewLinkGenerator(base *storage.Database, req models.InvitationCreateFewRequest, userId, url string) ([]models.Invitation, []string, error) {
	var (
		inviteReq   = req.Invitations
		org_id      = req.OrganisationID
		invitations []models.Invitation
		errs        []string
		user        models.User
	)

	for _, invite := range inviteReq {
		var existingInvite models.Invitation

		isTelexUser := postgresql.CheckExists(base.Postgresql, &user, "email = ?", invite.Email)

		// Check if the user's email has a pending invitation for that organisation with a pending status
		invitationExists := postgresql.CheckExists(base.Postgresql, &existingInvite, "email = ? AND organisation_id = ? AND status = 'invited' AND expires_at > ?", invite.Email, org_id, time.Now().UTC())
		if invitationExists {
			invitations = append(invitations, existingInvite)
			continue
		}

		if isTelexUser {
			alreadyMember := postgresql.CheckExists(base.Postgresql, &models.OrgUserManagement{}, "user_id = ? AND organisation_id = ?", user.ID, org_id)
			if alreadyMember {
				errs = append(errs, fmt.Errorf("%s is already a member of the organisation", invite.Email).Error())
				continue
			}
		}

		token, err := utility.GenerateInvitationToken()
		if err != nil {
			errs = append(errs, fmt.Sprintf("could not generate token for %s: %v", invite.Email, err))
			continue
		}

		ids := models.IDS{
			UserID:         userId,
			OrganisationID: org_id,
			RoleID:         invite.Role,
		}

		invitation := CreateInvitation(invite.Email, token, "invited", isTelexUser, ids)
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
			InvitedBy:      invite.InvitedBy,
			InvitationLink: utility.GenerateInvitationLink(baseURL, invite.OrganisationID, invite.Token),
			Sent_At:        invite.CreatedAt,
			Expires_At:     invite.ExpiresAt,
		})
	}
	return response
}

func AdminInvitationVerify(db *gorm.DB, req models.VerifyInvitationLinkRequest, logger *utility.Logger) {

}

func VerifyInvitation(req models.VerifyInvitationLinkRequest, db *storage.Database, c *gin.Context, logger *utility.Logger) (gin.H, int, error) {

	var (
		user         = models.User{}
		responseData gin.H
		i            = models.Invitation{}
		orgmgt       = models.OrgUserManagement{}
		userID       string
	)

	invitation, code, err := i.GetInvitationLinkByToken(db.Postgresql, req.Token, logger)
	if err != nil {
		logger.Error("Error getting invitation link by token", err)
		return responseData, code, err
	}

	user, err = getOrCreateUser(invitation, db.Postgresql)
	if err != nil {
		logger.Error("error in getting or creating user", err)
		return responseData, http.StatusInternalServerError, err
	}

	userID = user.ID
	invitation.Status = "accepted"

	err = invitation.UpdateInvitation(db.Postgresql)
	if err != nil {
		logger.Error("error in updating invitation on acceptance", err)
		return responseData, http.StatusBadRequest, err
	}

	orgmgt.RoleID = invitation.Role
	orgmgt.Status = "active"
	orgmgt.UserID = userID
	orgmgt.OrganisationID = invitation.OrganisationID

	err = addUserToOrganisation(orgmgt, db.Postgresql)
	if err != nil {
		logger.Error("error adding user to the invited organisation", err)
		return responseData, http.StatusInternalServerError, err
	}

	defaultChannel, err := getGeneralChannel(db.Postgresql, orgmgt.OrganisationID)
	if err != nil {
		logger.Error("error getting general channel", err)
	}

	if defaultChannel.ID != "" {
		err = addUserToChannel(&defaultChannel, orgmgt, logger, db)
		if err != nil {
			logger.Error("error adding user to the general channel", err)
			return responseData, http.StatusInternalServerError, err
		}
	}

	userData, err := user.GetUserByEmail(db.Postgresql, invitation.Email)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("unable to fetch user")
	}

	tokenData, err := middleware.CreateToken(userData, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error creating token")
	}

	err = saveAccessToken(tokenData, userID, db.Postgresql)
	if err != nil {
		return responseData, http.StatusInternalServerError, errors.New("error saving access token")
	}

	responseData = buildUserResponse(userData, tokenData)

	return responseData, http.StatusOK, nil
}

func getGeneralChannel(db *gorm.DB, orgID string) (models.Channels, error) {
	var channels models.Channels

	err := db.Where("organisation_id = ? AND name = ?", orgID, "general").First(&channels).Error
	if err == nil {
		return channels, nil
	}

	//if general not found, get the first channel
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Where("organisation_id = ?", orgID).
			Order("created_at ASC").
			First(&channels).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return channels, fmt.Errorf("no channel found in organisation: %v", err)
			}
			return channels, fmt.Errorf("error getting first channel: %v", err)
		}
	}

	if channels.ID == "" {
		return channels, errors.New("no channel found in organisation")
	}

	return channels, fmt.Errorf("database error: %w", err)
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
				ID:       utility.GenerateUUID(),
				UserName: name,
			},
		}

		err := user.CreateUser(db)
		if err != nil {
			return user, err
		}

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

	if _, err := orgmgt.AddUserToOrganisation(db); err != nil {
		return err
	}

	return nil
}

func addUserToChannel(chans *models.Channels, orgmgt models.OrgUserManagement, logger *utility.Logger, db *storage.Database) error {
	var profile models.Profile

	err := profile.GetProfileByUserId(db.Postgresql, orgmgt.UserID)

	if err != nil {
		return fmt.Errorf("failed to get user profile: %v", err)
	}

	reqs := models.JoinChannelsRequest{
		Username:   profile.UserName,
		ChannelsID: chans.ID,
		UserID:     orgmgt.UserID,
	}

	if _, err := chans.AddUserToChannel(db.Postgresql, reqs); err != nil {
		return err
	}
	systemMsg := models.CreateThreadMsgReq{
		Content:    fmt.Sprintf("%s joined this channel", profile.UserName),
		Type:       "system",
		UserId:     orgmgt.UserID,
		ChannelsID: chans.ID,
		OrgId:      chans.OrganisationID,
		ThreadId:   utility.GenerateUUID(),
	}

	_, err = thread.SaveThreadMessage(systemMsg, db, logger)

	if err != nil {
		logger.Error("failed to save system message for channel %s", chans.Name)
	}

	logger.Info("Added system message for user joining the channel")

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
