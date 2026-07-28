package buzz

import (
	"errors"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// SendBuzzReaction sends an ephemeral reaction in a buzz call (broadcast-only, not persisted)
func SendBuzzReaction(db *storage.Database, logger *utility.Logger, req models.SendBuzzReactionRequest, userID string) (int, error) {
	// Validate buzz is active
	buzz, err := permissions.IsBuzzActive(db.Postgresql, req.BuzzID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "send reaction")
		logger.Error("buzz validation failed: %v", err)
		return statusCode, errors.New(errMsg)
	}

	// Verify user is an active participant
	var participant models.BuzzParticipant
	if err := db.Postgresql.Where("buzz_id = ? AND user_id = ? AND status = ?",
		req.BuzzID, userID, models.BuzzParticipantStatusActive).First(&participant).Error; err != nil {
		logger.Error("user %s is not an active participant in buzz %s", userID, req.BuzzID)
		return http.StatusForbidden, errors.New("you must be an active participant to send reactions")
	}

	// Fetch user profile for username
	var profModel models.Profile
	orgID := ""
	if buzz.OrgID != nil {
		orgID = *buzz.OrgID
	}
	profile, err := profModel.GetOrCreateProfileForOrg(db.Postgresql, userID, orgID)
	if err != nil {
		logger.Error("failed to fetch user profile: %v", err)
		return http.StatusInternalServerError, errors.New("failed to fetch user details")
	}

	// Create ephemeral reaction event
	reactionEvent := models.BuzzReactionPayload{
		Event:        "buzz_reaction",
		BuzzID:       req.BuzzID,
		ChannelID:    buzz.ChannelID,
		UserID:       userID,
		UserName:     profile.UserName,
		ReactionType: req.ReactionType,
		Content:      req.Content,
		Timestamp:    time.Now().UTC(),
	}

	// Publish to channel for all participants to see
	notification := models.Notification[models.BuzzReactionEvent]
	notification.SectionType = models.ChannelsSection
	notification.Content = reactionEvent
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()
	buzzChannelId := getPublishChannel(buzz)

	if err := centrifuge.PublishChannelOptional(logger, buzzChannelId, notification); err != nil {
		logger.Warning("failed to publish buzz reaction event (non-critical): %v", err)
	}

	logger.Info("user %s sent %s reaction '%s' in buzz %s", userID, req.ReactionType, req.Content, req.BuzzID)
	return http.StatusOK, nil
}

// UpdateBuzzSticker updates a participant's status sticker and broadcasts the change
func UpdateBuzzSticker(db *storage.Database, logger *utility.Logger, req models.BuzzStickerUpdateRequest, userID string) (models.BuzzStickerUpdateResponse, int, error) {
	var resp models.BuzzStickerUpdateResponse

	// Validate that user is an active participant in the buzz
	buzz, err := permissions.IsBuzzActive(db.Postgresql, req.BuzzID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "update sticker")
		logger.Error("buzz validation failed: %v", err)
		return resp, statusCode, errors.New(errMsg)
	}

	// Fetch user profile early to fail fast before any DB writes
	var profModel models.Profile
	orgID := ""
	if buzz.OrgID != nil {
		orgID = *buzz.OrgID
	}
	profile, err := profModel.GetOrCreateProfileForOrg(db.Postgresql, userID, orgID)
	if err != nil {
		logger.Error("failed to fetch user profile: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch user details")
	}

	var participant models.BuzzParticipant
	if err := db.Postgresql.Where("buzz_id = ? AND user_id = ? AND status = ?",
		req.BuzzID, userID, models.BuzzParticipantStatusActive).First(&participant).Error; err != nil {
		logger.Error("user %s is not an active participant in buzz %s", userID, req.BuzzID)
		return resp, http.StatusForbidden, errors.New("you must be an active participant to update stickers")
	}

	// Prepare sticker update
	now := time.Now().UTC()
	var stickerSetAt *time.Time
	if req.Sticker != nil {
		stickerSetAt = &now
	}

	// Update participant's sticker in database
	updates := map[string]interface{}{
		"status_sticker": req.Sticker,
		"sticker_set_at": stickerSetAt,
		"updated_at":     now,
	}

	if err := db.Postgresql.Model(&models.BuzzParticipant{}).
		Where("buzz_id = ? AND user_id = ?", req.BuzzID, userID).
		Updates(updates).Error; err != nil {
		logger.Error("failed to update participant sticker: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to update sticker")
	}

	// Broadcast sticker update via Centrifuge
	stickerEvent := models.BuzzStickerPayload{
		Event:        "buzz_sticker_update",
		BuzzID:       req.BuzzID,
		ChannelID:    buzz.ChannelID,
		UserID:       userID,
		UserName:     profile.UserName,
		Sticker:      req.Sticker,
		StickerSetAt: stickerSetAt,
		Timestamp:    now,
	}

	// Publish to channel for all participants to see
	notification := models.Notification[models.BuzzStickerEvent]
	notification.SectionType = models.ChannelsSection
	notification.Content = stickerEvent
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	buzzChannelId := getPublishChannel(buzz)

	if err := centrifuge.PublishChannelOptional(logger, buzzChannelId, notification); err != nil {
		logger.Warning("failed to publish buzz sticker event (non-critical): %v", err)
	}

	resp = models.BuzzStickerUpdateResponse{
		BuzzID:       req.BuzzID,
		UserID:       userID,
		Sticker:      req.Sticker,
		StickerSetAt: stickerSetAt,
		UpdatedAt:    now,
	}

	logger.Info("user %s updated sticker to %v in buzz %s", userID, req.Sticker, req.BuzzID)
	return resp, http.StatusOK, nil
}
