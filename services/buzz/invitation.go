package buzz

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/actions/names"
	"github.com/hngprojects/telex_be/utility"
)

// SearchChannelMembers searches for channel members that can be invited to a buzz
func SearchChannelMembers(db *storage.Database, logger *utility.Logger, req models.SearchChannelMembersRequest, userID string) (models.SearchChannelMembersResponse, int, error) {
	var resp models.SearchChannelMembersResponse

	buzz, err := permissions.IsBuzzActive(db.Postgresql, req.BuzzID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "search members")
		logger.Error("buzz validation failed: %v", err)
		return resp, statusCode, errors.New(errMsg)
	}

	if !models.IsUserInChannel(db.Postgresql, req.ChannelID, userID) {
		return resp, http.StatusForbidden, errors.New("you are not a member of this channel")
	}

	limit := req.Limit
	if limit == 0 {
		limit = 50
	}

	query := db.Postgresql.Table("user_channels uc").
		Select(`p.userid as user_id,
				p.user_name as user_name,
				CONCAT(p.first_name, ' ', p.last_name) as full_name,
				u.email,
				p.avatar_url as avatar_url`).
		Joins("JOIN profiles p ON p.userid = uc.user_id").
		Joins("JOIN users u ON u.id = uc.user_id").
		Where("uc.channels_id = ?", req.ChannelID)

	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		query = query.Where(
			"LOWER(p.user_name) LIKE ? OR LOWER(p.first_name) LIKE ? OR LOWER(p.last_name) LIKE ? OR LOWER(u.email) LIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	var members []models.ChannelMemberInfo
	if err := query.Limit(limit).Scan(&members).Error; err != nil {
		logger.Error("failed to search channel members: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to search members")
	}

	buzzParticipants := make(map[string]bool)
	for _, participantID := range buzz.ParticipantIDs {
		buzzParticipants[participantID] = true
	}

	var pendingInvitations []models.BuzzInvitation
	if err := db.Postgresql.Where("buzz_id = ? AND status = ?", req.BuzzID, models.BuzzInvitationPending).
		Find(&pendingInvitations).Error; err != nil {
		logger.Error("failed to fetch pending invitations: %v", err)
	}

	invitedUsers := make(map[string]bool)
	for _, inv := range pendingInvitations {
		invitedUsers[inv.InviteeID] = true
	}

	for i := range members {
		members[i].IsInBuzz = buzzParticipants[members[i].UserID]
		members[i].IsInvited = invitedUsers[members[i].UserID]
	}

	resp.Members = members
	resp.Total = len(members)

	return resp, http.StatusOK, nil
}

// InviteUsersToBuzz invites multiple users to a buzz
func InviteUsersToBuzz(db *storage.Database, logger *utility.Logger, req models.InviteUsersToBuzzRequest, inviterID string) (models.InviteUsersToBuzzResponse, int, error) {
	var resp models.InviteUsersToBuzzResponse

	buzz, err := permissions.IsBuzzActive(db.Postgresql, req.BuzzID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "invite users")
		logger.Error("buzz validation failed: %v", err)
		return resp, statusCode, errors.New(errMsg)
	}

	_, err = permissions.CanPerformBuzzAction(db.Postgresql, req.BuzzID, inviterID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "invite users")
		logger.Error("inviter permission check failed: %v", err)
		return resp, statusCode, errors.New(errMsg)
	}

	var inviterProfile models.Profile
	if err := db.Postgresql.Where("userid = ?", inviterID).First(&inviterProfile).Error; err != nil {
		logger.Error("failed to fetch inviter profile: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch inviter profile")
	}

	now := time.Now().UTC()
	var successfulInvites []string
	var failedInvites []string

	for _, inviteeID := range req.InviteeIDs {
		isInBuzz := false
		for _, participantID := range buzz.ParticipantIDs {
			if participantID == inviteeID {
				isInBuzz = true
				break
			}
		}
		if isInBuzz {
			logger.Info("user %s is already in buzz %s, skipping", inviteeID, req.BuzzID)
			failedInvites = append(failedInvites, inviteeID)
			continue
		}

		if buzz.BuzzType != models.BuzzTypeOrganization {
			if !models.IsUserInChannel(db.Postgresql, buzz.ChannelID, inviteeID) {
				logger.Info("user %s is not a channel member, skipping", inviteeID)
				failedInvites = append(failedInvites, inviteeID)
				continue
			}
		}

		exists, err := models.CheckInvitationExists(db.Postgresql, req.BuzzID, inviteeID)
		if err != nil {
			logger.Error("failed to check invitation existence: %v", err)
			failedInvites = append(failedInvites, inviteeID)
			continue
		}
		if exists {
			logger.Info("pending invitation already exists for user %s", inviteeID)
			successfulInvites = append(successfulInvites, inviteeID)
			continue
		}

		invitation := models.BuzzInvitation{
			ID:        utility.GenerateUUID(),
			BuzzID:    req.BuzzID,
			ChannelID: buzz.ChannelID,
			InviterID: inviterID,
			InviteeID: inviteeID,
			Status:    models.BuzzInvitationPending,
			InvitedAt: now,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := db.Postgresql.Create(&invitation).Error; err != nil {
			logger.Error("failed to create invitation for user %s: %v", inviteeID, err)
			failedInvites = append(failedInvites, inviteeID)
			continue
		}

		if err := publishInvitationEvent(logger, invitation, inviterProfile.UserName); err != nil {
			logger.Error("failed to publish invitation event: %v", err)
		}

		// Send buzz invitation email asynchronously
		go sendBuzzInvitationEmail(db, logger, inviteeID, inviterProfile.UserName, buzz.ID)

		successfulInvites = append(successfulInvites, inviteeID)
	}

	resp.BuzzID = req.BuzzID
	resp.InvitedUserIDs = successfulInvites
	resp.FailedUserIDs = failedInvites
	resp.InvitationsSent = len(successfulInvites)

	if len(successfulInvites) == 0 {
		resp.Message = "no invitations were sent"
		return resp, http.StatusBadRequest, errors.New("failed to send any invitations")
	}

	if len(failedInvites) > 0 {
		resp.Message = fmt.Sprintf("sent %d invitation(s), %d failed", len(successfulInvites), len(failedInvites))
	} else {
		resp.Message = fmt.Sprintf("successfully sent %d invitation(s)", len(successfulInvites))
	}

	return resp, http.StatusCreated, nil
}

// RespondToInvitation allows a user to accept or decline a buzz invitation
func RespondToInvitation(db *storage.Database, logger *utility.Logger, req models.RespondToInvitationRequest, userID string) (models.RespondToInvitationResponse, int, error) {
	var resp models.RespondToInvitationResponse

	var invitation models.BuzzInvitation
	if err := db.Postgresql.Where("id = ?", req.InvitationID).First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("invitation not found")
		}
		logger.Error("failed to fetch invitation: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch invitation")
	}

	if invitation.InviteeID != userID {
		return resp, http.StatusForbidden, errors.New("you are not authorized to respond to this invitation")
	}

	if invitation.Status != models.BuzzInvitationPending {
		return resp, http.StatusBadRequest, errors.New("invitation has already been responded to")
	}

	_, err := permissions.IsBuzzActive(db.Postgresql, invitation.BuzzID)
	if err != nil {
		if err == permissions.ErrBuzzEnded {
			now := time.Now().UTC()
			invitation.Status = models.BuzzInvitationExpired
			invitation.RespondedAt = &now
			db.Postgresql.Save(&invitation)
			return resp, http.StatusGone, errors.New("buzz has ended")
		}
		statusCode, errMsg := mapPermissionError(err, "respond to invitation")
		return resp, statusCode, errors.New(errMsg)
	}

	now := time.Now().UTC()
	invitation.RespondedAt = &now

	var inviteeProfile models.Profile
	if err := db.Postgresql.Where("userid = ?", userID).First(&inviteeProfile).Error; err != nil {
		logger.Error("failed to fetch invitee profile: %v", err)
	}

	if req.Accept {
		invitation.Status = models.BuzzInvitationAccepted

		if err := db.Postgresql.Save(&invitation).Error; err != nil {
			logger.Error("failed to update invitation status: %v", err)
			return resp, http.StatusInternalServerError, errors.New("failed to update invitation")
		}

		joinResp, statusCode, err := JoinBuzz(db, logger, invitation.BuzzID, userID)
		if err != nil {
			logger.Error("failed to join buzz after accepting invitation: %v", err)
			return resp, statusCode, err
		}

		resp.InvitationID = invitation.ID
		resp.BuzzID = invitation.BuzzID
		resp.Status = models.BuzzInvitationAccepted
		resp.Message = "invitation accepted and joined buzz"
		resp.AgoraToken = joinResp.AgoraToken

		publishInvitationResponseEvent(logger, invitation, inviteeProfile.UserName, "buzz_invitation_accepted")

	} else {
		invitation.Status = models.BuzzInvitationDeclined

		if err := db.Postgresql.Save(&invitation).Error; err != nil {
			logger.Error("failed to update invitation status: %v", err)
			return resp, http.StatusInternalServerError, errors.New("failed to update invitation")
		}

		resp.InvitationID = invitation.ID
		resp.BuzzID = invitation.BuzzID
		resp.Status = models.BuzzInvitationDeclined
		resp.Message = "invitation declined"

		publishInvitationResponseEvent(logger, invitation, inviteeProfile.UserName, "buzz_invitation_declined")
	}

	return resp, http.StatusOK, nil
}

// GetPendingInvitations retrieves all pending invitations for a user and transforms them to response format
func GetPendingInvitations(db *storage.Database, logger *utility.Logger, userID string) ([]models.BuzzInvitationResponse, int, error) {
	var responses []models.BuzzInvitationResponse

	var invitations []models.BuzzInvitation
	if err := db.Postgresql.Preload("Inviter").Where("invitee_id = ?", userID).Find(&invitations).Error; err != nil {
		logger.Error("failed to fetch pending invitations: %v", err)
		return responses, http.StatusInternalServerError, errors.New("failed to fetch invitations")
	}

	// Transform to response format
	for _, inv := range invitations {
		responses = append(responses, models.BuzzInvitationResponse{
			InvitationID: inv.ID,
			BuzzID:       inv.BuzzID,
			ChannelID:    inv.ChannelID,
			InviterID:    inv.InviterID,
			InviterName:  inv.Inviter.UserName,
			Status:       inv.Status,
			InvitedAt:    inv.InvitedAt,
		})
	}

	return responses, http.StatusOK, nil
}

// publishInvitationEvent sends a real-time notification to the invitee
func publishInvitationEvent(logger *utility.Logger, invitation models.BuzzInvitation, inviterName string) error {
	eventPayload := models.BuzzInvitationEventPayload{
		Event:        "buzz_invitation_received",
		InvitationID: invitation.ID,
		BuzzID:       invitation.BuzzID,
		ChannelID:    invitation.ChannelID,
		InviterID:    invitation.InviterID,
		InviterName:  inviterName,
		InviteeID:    invitation.InviteeID,
		InvitedAt:    invitation.InvitedAt,
	}

	notification := models.Notification[models.NotificationType("buzz_invitation_received")]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: invitation.ChannelID,
		UserId:    invitation.InviteeID,
	}
	notification.NotificationId = utility.GenerateUUID()

	return centrifuge.PublishChannel(logger, "user:"+invitation.InviteeID, notification)
}

// publishInvitationResponseEvent notifies the channel about invitation response
func publishInvitationResponseEvent(logger *utility.Logger, invitation models.BuzzInvitation, inviteeName, eventType string) {
	eventPayload := models.BuzzInvitationResponseEventPayload{
		Event:        eventType,
		InvitationID: invitation.ID,
		BuzzID:       invitation.BuzzID,
		ChannelID:    invitation.ChannelID,
		InviteeID:    invitation.InviteeID,
		InviteeName:  inviteeName,
		Status:       invitation.Status,
		RespondedAt:  time.Now().UTC(),
	}

	notification := models.Notification[models.NotificationType(eventType)]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: invitation.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, invitation.ChannelID, notification); err != nil {
		logger.Error("failed to publish invitation response event: %v", err)
	}
}

// sendBuzzInvitationEmail sends a buzz invitation email to a user asynchronously via Redis queue
func sendBuzzInvitationEmail(db *storage.Database, logger *utility.Logger, inviteeID, inviterName, buzzID string) {
	var inviteeUser models.User
	if err := db.Postgresql.Where("id = ?", inviteeID).First(&inviteeUser).Error; err != nil {
		logger.Error("failed to fetch invitee user for email: %v", err)
		return
	}

	var inviteeProfile models.Profile
	if err := db.Postgresql.Where("userid = ?", inviteeID).First(&inviteeProfile).Error; err != nil {
		logger.Error("failed to fetch invitee profile for email: %v", err)
		return
	}

	buzzCode := utility.ExtractBuzzCode(buzzID)
	configData := config.GetConfig()
	joinLink := fmt.Sprintf("%s/client/buzz/%s", configData.App.FRONTEND_URL, buzzCode)

	inviteeName := inviteeProfile.UserName
	if inviteeProfile.FirstName != "" {
		inviteeName = inviteeProfile.FirstName
	}

	reqData := models.SendBuzzInvitationEmail{
		Email:       inviteeUser.Email,
		InviteeName: inviteeName,
		InviterName: inviterName,
		BuzzCode:    buzzCode,
		JoinLink:    joinLink,
	}

	if err := actions.AddNotificationToQueue(storage.DB.Redis, names.SendBuzzInvitationEmail, reqData); err != nil {
		logger.Error("failed to enqueue buzz invitation email for user %s: %v", inviteeID, err)
	}
}
