package buzz

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	"github.com/hngprojects/telex_be/utility"
)

const (
	errorAgoraNotInitialized = "agora service not initialized"
)

// mapPermissionError maps permission errors to HTTP status codes and user-friendly messages
func mapPermissionError(err error, action string) (int, string) {
	switch err {
	case permissions.ErrBuzzNotFound:
		return http.StatusNotFound, "buzz does not exist"
	case permissions.ErrBuzzEnded:
		return http.StatusConflict, "buzz has ended"
	case permissions.ErrNotChannelMember:
		return http.StatusForbidden, "user is not a member of the channel"
	case permissions.ErrAlreadyInBuzz:
		return http.StatusBadRequest, "user is already in the buzz"
	case permissions.ErrNotHost:
		return http.StatusForbidden, "only the buzz host can perform this action"
	case permissions.ErrNotActiveParticipant:
		return http.StatusForbidden, "you must be an active participant"
	case permissions.ErrChannelNotFound:
		return http.StatusNotFound, "channel does not exist"
	case permissions.ErrBuzzAlreadyActive:
		return http.StatusConflict, "channel already has an active buzz"
	default:
		return http.StatusInternalServerError, fmt.Sprintf("failed to validate %s permissions", action)
	}
}

// getParticipantsMetadata fetches detailed metadata for all participants in a buzz
func getParticipantsMetadata(db *gorm.DB, buzzID string) ([]models.ParticipantMetadata, error) {
	var participants []models.ParticipantMetadata

	query := `
		SELECT
			bp.user_id,
			COALESCE(NULLIF(TRIM(p.user_name), ''), p.first_name, '') as user_name,
			COALESCE(
				NULLIF(TRIM(p.full_name), ''), 
				CASE 
					WHEN TRIM(p.last_name) IS NOT NULL AND TRIM(p.last_name) != '' 
					THEN CONCAT(TRIM(p.first_name), ' ', TRIM(p.last_name))
					ELSE TRIM(p.first_name)
				END,
				''
			) as full_name,
			p.avatar_url,
			bp.joined_at,
			bp.status,
			bp.status_sticker,
			bp.sticker_set_at
		FROM buzz_participants bp
		JOIN users u ON bp.user_id = u.id
		LEFT JOIN profiles p ON u.id = p.userid
		WHERE bp.buzz_id = ?
		ORDER BY bp.joined_at ASC
	`

	if err := db.Raw(query, buzzID).Scan(&participants).Error; err != nil {
		return nil, err
	}

	return participants, nil
}

// buildBuzzMetadataResponse builds the base metadata response for a buzz
func buildBuzzMetadataResponse(buzz *models.Buzz, participantMetadata []models.ParticipantMetadata) models.BuzzMetadataResponse {
	return models.BuzzMetadataResponse{
		BuzzID:       buzz.ID,
		HostID:       buzz.HostID,
		ChannelID:    buzz.ChannelID,
		Status:       buzz.Status,
		CreatedAt:    buzz.CreatedAt,
		StartedAt:    buzz.BuzzStartTime,
		Participants: participantMetadata,
	}
}

// CreateBuzz creates a new buzz, adds the host as the first participant, and emits a realtime event.
func CreateBuzz(db *storage.Database, logger *utility.Logger, req models.CreateBuzzRequest, hostID string) (models.BuzzCreateResponse, int, error) {
	var resp models.BuzzCreateResponse
	var channelID string

	// Handle DM channel creation if participant_id is provided
	if req.ParticipantID != nil && *req.ParticipantID != "" {
		logger.Info("creating buzz with new DM channel for user %s and participant %s", hostID, *req.ParticipantID)

		// Get user's organization
		var user models.User
		if err := db.Postgresql.Where("id = ?", hostID).First(&user).Error; err != nil {
			logger.Error("failed to get user for DM creation: %v", err)
			return resp, http.StatusBadRequest, errors.New("user not found")
		}

		// Check if DM channel already exists
		var existingDM models.DmChannels
		exists := db.Postgresql.Where("(user_id = ? AND participant_id = ?) OR (user_id = ? AND participant_id = ?)",
			hostID, *req.ParticipantID, *req.ParticipantID, hostID).First(&existingDM).Error == nil

		if exists {
			// Use existing DM channel
			channelID = existingDM.ChannelId
			logger.Info("using existing DM channel %s", channelID)
		} else {
			// Create new DM channel
			dmReq := models.DmChannelsRequest{
				UserId:        hostID,
				ParticipantId: *req.ParticipantID,
				OrgId:         user.CurrentOrg.String(),
				ChatType:      "user",
			}

			dmResp, code, err := dm.CreateDmChannel(dmReq, request.ExternalRequest{Logger: logger, Test: false}, db, logger)
			if err != nil {
				logger.Error("failed to create DM channel: %v", err)
				return resp, code, errors.New("failed to create DM channel for buzz")
			}
			channelID = dmResp.ID
			logger.Info("created new DM channel %s", channelID)
		}
	} else if req.ChannelID != "" {
		channelID = req.ChannelID
	} else {
		return resp, http.StatusBadRequest, errors.New("either channel_id or participant_id must be provided")
	}

	logger.Info("creating buzz for user %s in channel %s", hostID, channelID)

	// Validate permissions using centralized permission check
	err := permissions.CanCreateBuzz(db.Postgresql, channelID, hostID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "create")
		logger.Error("buzz creation failed - permission denied for user %s in channel %s: %v", hostID, req.ChannelID, err)
		return resp, statusCode, errors.New(errMsg)
	}
	logger.Info("permission validated for user %s to create buzz in channel %s", hostID, channelID)

	// Determine channel type (regular, DM, or group DM)
	channelType, err := permissions.GetChannelType(db.Postgresql, channelID)
	if err != nil {
		logger.Error("failed to determine channel type for channel %s: %v", req.ChannelID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to determine channel type")
	}
	logger.Info("determined channel type: %s for channel %s", channelType, channelID)

	// Fail on Agora service not available before permission checks
	service := agora.Client.Service
	if service == nil {
		logger.Error(errorAgoraNotInitialized)
		return resp, http.StatusInternalServerError, errors.New(errorAgoraNotInitialized)
	}

	now := time.Now().UTC()
	// Only the host auto-joins on creation; others must explicitly join via /join endpoint
	participants := []string{hostID}

	buzz := models.Buzz{
		ID:             utility.GenerateUUID(),
		ChannelID:      channelID,
		ChannelType:    channelType,
		HostID:         hostID,
		ParticipantIDs: participants,
		BuzzStartTime:  now,
		IsLiveStatus:   true,
		Status:         models.BuzzStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Generate Agora token BEFORE creating buzz in database (using hostID as UID)
	// Use constant for token expiration (4 hours)
	logger.Info("generating Agora RTC token for host %s in buzz %s", hostID, buzz.ID)
	token, err := service.GenerateRTCToken(buzz.ID, hostID, hostID, agora.DefaultTokenExpirationSeconds)
	if err != nil {
		logger.Error("buzz creation failed - Agora token generation error for host %s in buzz %s: %v", hostID, buzz.ID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	agoraToken := models.BuzzAgoraTokenResponse{
		Token:       token,
		AppId:       service.GetAppId(),
		ChannelName: buzz.ID,
		UID:         hostID,
	}

	logger.Info("starting database transaction to create buzz %s", buzz.ID)
	err = db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&buzz).Error; err != nil {
			logger.Error("transaction failed - buzz record creation error: %v", err)
			return err
		}

		// Create participant record for the host only
		hostParticipant := models.BuzzParticipant{
			ID:       utility.GenerateUUID(),
			BuzzID:   buzz.ID,
			UserID:   hostID,
			Status:   models.BuzzParticipantStatusActive,
			IsMuted:  false,
			JoinedAt: now,
		}

		if err := tx.Create(&hostParticipant).Error; err != nil {
			logger.Error("transaction failed - buzz participant creation error: %v", err)
			return err
		}

		return nil
	})
	if err != nil {
		logger.Error("buzz creation failed - database transaction error for buzz %s: %v", buzz.ID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to create buzz")
	}

	// Fetch participant metadata
	participantMetadata, err := getParticipantsMetadata(db.Postgresql, buzz.ID)
	if err != nil {
		logger.Error("failed to fetch participant metadata: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch participant details")
	}

	metadataResp := buildBuzzMetadataResponse(&buzz, participantMetadata)
	resp = models.BuzzCreateResponse{
		BuzzID:         metadataResp.BuzzID,
		HostID:         metadataResp.HostID,
		ChannelID:      metadataResp.ChannelID,
		Status:         metadataResp.Status,
		CreatedAt:      metadataResp.CreatedAt,
		StartedAt:      metadataResp.StartedAt,
		ParticipantIDs: buzz.ParticipantIDs,
		Participants:   metadataResp.Participants,
		AgoraToken:     &agoraToken,
	}

	eventPayload := models.BuzzEventPayload{
		Event:          string(models.BuzzStarted),
		BuzzID:         buzz.ID,
		ChannelID:      buzz.ChannelID,
		HostID:         buzz.HostID,
		ParticipantIDs: participants,
		CreatedAt:      buzz.BuzzStartTime,
		Status:         buzz.Status,
	}

	notification := models.Notification[models.BuzzStarted]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, buzz.ChannelID, notification); err != nil {
		logger.Error("failed to publish buzz event: %v", err)
	}

	logger.Info("buzz %s created successfully by user %s in channel %s (type: %s)", buzz.ID, hostID, req.ChannelID, channelType)
	return resp, http.StatusCreated, nil
}

// JoinBuzz allows a user to join an existing buzz
func JoinBuzz(db *storage.Database, logger *utility.Logger, buzzID string, userID string) (models.JoinBuzzResponse, int, error) {
	var resp models.JoinBuzzResponse

	logger.Info("user %s attempting to join buzz %s", userID, buzzID)

	// Fail-fast: Check Agora service availability before permission checks
	service := agora.Client.Service
	if service == nil {
		logger.Error(errorAgoraNotInitialized)
		return resp, http.StatusInternalServerError, errors.New(errorAgoraNotInitialized)
	}

	// Validate all join permissions using centralized permission check
	buzz, err := permissions.CanJoinBuzz(db.Postgresql, buzzID, userID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "join")
		logger.Error("join buzz failed - permission denied for user %s in buzz %s: %v", userID, buzzID, err)
		return resp, statusCode, errors.New(errMsg)
	}
	logger.Info("permission validated for user %s to join buzz %s", userID, buzzID)

	timestamp := time.Now().UTC()

	// Generate Agora token BEFORE adding user to buzz (using userID as UID)
	// This way if token generation fails, we haven't polluted the database
	// Use constant for token expiration (4 hours)
	logger.Info("generating Agora RTC token for user %s in buzz %s", userID, buzzID)
	token, err := service.GenerateRTCToken(buzzID, userID, userID, agora.DefaultTokenExpirationSeconds)
	if err != nil {
		logger.Error("join buzz failed - Agora token generation error for user %s in buzz %s: %v", userID, buzzID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	agoraToken := models.BuzzAgoraTokenResponse{
		Token:       token,
		AppId:       service.GetAppId(),
		ChannelName: buzzID,
		UID:         userID,
	}

	// Add user to buzz in transaction
	logger.Info("starting database transaction to add user %s to buzz %s", userID, buzzID)
	if err := addUserToBuzzTransaction(db, logger, buzz, userID, timestamp); err != nil {
		logger.Error("join buzz failed - database transaction error for user %s in buzz %s: %v", userID, buzzID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to join buzz")
	}

	// Fetch updated buzz with new participant using fresh variable to avoid pointer issues
	var updatedBuzz models.Buzz
	if err := db.Postgresql.Where("id = ?", buzzID).First(&updatedBuzz).Error; err != nil {
		logger.Error("failed to fetch updated buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch updated buzz")
	}
	buzz = &updatedBuzz

	// Fetch participant metadata
	participantMetadata, err := getParticipantsMetadata(db.Postgresql, buzzID)
	if err != nil {
		logger.Error("failed to fetch participant metadata: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch participant details")
	}

	metadataResp := buildBuzzMetadataResponse(buzz, participantMetadata)
	resp = models.JoinBuzzResponse{
		BuzzID:       metadataResp.BuzzID,
		HostID:       metadataResp.HostID,
		ChannelID:    metadataResp.ChannelID,
		UserID:       userID,
		Status:       metadataResp.Status,
		CreatedAt:    metadataResp.CreatedAt,
		JoinedAt:     timestamp,
		Participants: metadataResp.Participants,
		AgoraToken:   &agoraToken,
	}

	publishJoinBuzzEvent(logger, *buzz, timestamp)

	logger.Info("user %s successfully joined buzz %s", userID, buzzID)
	return resp, http.StatusOK, nil
}

func validateLeaveBuzz(db *storage.Database, logger *utility.Logger, buzzID, userID string) (models.Buzz, int, error) {
	var buzz models.Buzz

	// Use CanPerformBuzzAction to validate: buzz is active AND user is an active participant
	activeBuzz, err := permissions.CanPerformBuzzAction(db.Postgresql, buzzID, userID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "leave")
		logger.Error("permission check failed for user %s leaving buzz %s: %v", userID, buzzID, err)
		return buzz, statusCode, errors.New(errMsg)
	}

	buzz = *activeBuzz

	return buzz, http.StatusOK, nil
}

func addUserToBuzzTransaction(db *storage.Database, logger *utility.Logger, buzz *models.Buzz, userID string, timestamp time.Time) error {
	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := buzz.AddUserToBuzz(tx, userID); err != nil {
			return err
		}

		var existingParticipant models.BuzzParticipant
		err := tx.Where("buzz_id = ? AND user_id = ?", buzz.ID, userID).First(&existingParticipant).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				participant := models.BuzzParticipant{
					ID:       utility.GenerateUUID(),
					BuzzID:   buzz.ID,
					UserID:   userID,
					Status:   models.BuzzParticipantStatusActive,
					IsMuted:  false,
					JoinedAt: timestamp,
				}
				return tx.Create(&participant).Error
			}
			return err
		}

		existingParticipant.Status = models.BuzzParticipantStatusActive
		existingParticipant.IsMuted = false
		existingParticipant.JoinedAt = timestamp
		existingParticipant.LeftAt = nil
		return tx.Save(&existingParticipant).Error
	})

	if err != nil {
		logger.Error("failed to add user to buzz: %v", err)
		return errors.New("could not update buzz participants")
	}

	return nil
}

func publishJoinBuzzEvent(logger *utility.Logger, buzz models.Buzz, timestamp time.Time) {
	eventPayload := models.BuzzEventPayload{
		Event:          string(models.UserJoinedBuzz),
		BuzzID:         buzz.ID,
		ChannelID:      buzz.ChannelID,
		HostID:         buzz.HostID,
		ParticipantIDs: buzz.ParticipantIDs,
		CreatedAt:      timestamp,
		Status:         buzz.Status,
	}

	notification := models.Notification[models.UserJoinedBuzz]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, buzz.ChannelID, notification); err != nil {
		logger.Error("failed to publish join buzz event: %v", err)
	}
}

func LeaveBuzz(db *storage.Database, logger *utility.Logger, buzzID, userID string) (*models.BuzzLeaveResponse, int, error) {
	var (
		profile   models.Profile
		newHostID = ""
		buzzEnded = false
	)

	logger.Info("user %s attempting to leave buzz %s", userID, buzzID)

	buzz, status, err := validateLeaveBuzz(db, logger, buzzID, userID)
	if err != nil {
		return nil, status, err
	}
	logger.Info("validation passed for user %s to leave buzz %s", userID, buzzID)

	tx := db.Postgresql.Begin()
	if tx.Error != nil {
		logger.Error("Failed to begin transaction: %v", tx.Error)
		return nil, http.StatusInternalServerError, errors.New("failed to start transaction")
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error("Transaction panicked: %v", r)
		}
	}()

	newParticipants := make([]string, 0, len(buzz.ParticipantIDs)-1)
	for _, id := range buzz.ParticipantIDs {
		if id != userID {
			newParticipants = append(newParticipants, id)
		}
	}

	if userID == buzz.HostID && len(newParticipants) > 0 {
		buzz.HostID = newParticipants[0]
		newHostID = newParticipants[0]
	}
	// no participant left - end call
	if len(newParticipants) == 0 {
		now := time.Now().UTC()
		buzz.BuzzEndTime = &now
		buzz.Status = models.BuzzStatusEnded
		buzz.IsLiveStatus = false
		buzzEnded = true
	}

	buzz.ParticipantIDs = newParticipants

	if err = tx.Model(&buzz).Updates(buzz).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to update buzz: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to update buzz")
	}

	if err = tx.Model(&models.BuzzParticipant{}).
		Where("buzz_id = ? AND user_id = ?", buzzID, userID).
		Update("status", models.BuzzParticipantStatusLeft).Error; err != nil {
		tx.Rollback()
		logger.Error("Failed to update participant status: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to update participant")
	}

	if err := tx.Where("userid = ?", userID).First(&profile).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, errors.New("user not found")
		}
		logger.Error("Failed to fetch user profile: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to fetch user profile")
	}

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to commit changes")
	}

	publishPayload := models.BuzzLeaveEventPayload{
		HuddleStatus: buzz.Status,
		HostChanged:  !(newHostID == ""),
		UserID:       userID,
		UserName:     profile.UserName,
		BuzzEventPayload: models.BuzzEventPayload{
			Event:          string(models.UserLeftBuzz),
			BuzzID:         buzzID,
			HostID:         buzz.HostID,
			ParticipantIDs: newParticipants,
			Status:         models.BuzzStatusActive,
		},
	}

	centrifuge.PublishLeaveBuzzEvent(logger, buzz.ChannelID, buzzID, publishPayload)

	if buzzEnded {
		if err := models.ExpireInvitationsForBuzz(db.Postgresql, buzzID); err != nil {
			logger.Error("failed to expire invitations for buzz %s: %v", buzzID, err)
		}
	}

	if newHostID != "" {
		logger.Info("user %s left buzz %s - host transferred to %s", userID, buzzID, newHostID)
	} else if buzzEnded {
		logger.Info("user %s left buzz %s - buzz ended (no participants remaining)", userID, buzzID)
	} else {
		logger.Info("user %s successfully left buzz %s", userID, buzzID)
	}

	return &models.BuzzLeaveResponse{
		BuzzID:        buzzID,
		ParticipantID: userID,
		NewHostID:     newHostID,
		LeftAt:        time.Now().UTC(),
		BuzzEnded:     buzzEnded,
	}, http.StatusOK, nil
}

// EndBuzz allows the host to explicitly end a buzz for all participants
func EndBuzz(db *storage.Database, logger *utility.Logger, buzzID, hostID string) (*models.BuzzEndResponse, int, error) {
	logger.Info("host %s attempting to end buzz %s", hostID, buzzID)

	// Use centralized permission check
	buzz, err := permissions.CanPerformHostAction(db.Postgresql, buzzID, hostID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "end")
		logger.Error("permission check failed for user %s ending buzz %s: %v", hostID, buzzID, err)
		return nil, statusCode, errors.New(errMsg)
	}
	logger.Info("permission validated for host %s to end buzz %s", hostID, buzzID)

	now := time.Now().UTC()

	// Update buzz status in transaction
	err = db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&buzz).Updates(map[string]interface{}{
			"status":         models.BuzzStatusEnded,
			"is_live_status": false,
			"buzz_end_time":  &now,
			"updated_at":     now,
		}).Error; err != nil {
			return err
		}

		// Update all active participants to left status
		if err := tx.Model(&models.BuzzParticipant{}).
			Where("buzz_id = ? AND status = ?", buzzID, models.BuzzParticipantStatusActive).
			Updates(map[string]interface{}{
				"status":  models.BuzzParticipantStatusLeft,
				"left_at": now,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("failed to end buzz: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to end buzz")
	}

	// Expire all pending invitations for this buzz
	if err := models.ExpireInvitationsForBuzz(db.Postgresql, buzzID); err != nil {
		logger.Error("failed to expire invitations for buzz %s: %v", buzzID, err)
	}

	// Publish buzz ended event
	eventPayload := models.BuzzEventPayload{
		Event:          string(models.BuzzEnded),
		BuzzID:         buzz.ID,
		ChannelID:      buzz.ChannelID,
		HostID:         buzz.HostID,
		ParticipantIDs: []string{}, // Empty since all participants have left
		CreatedAt:      now,
		Status:         buzz.Status,
	}

	notification := models.Notification[models.BuzzEnded]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, buzz.ChannelID, notification); err != nil {
		logger.Error("failed to publish buzz ended event: %v", err)
	}

	resp := &models.BuzzEndResponse{
		BuzzID:    buzz.ID,
		ChannelID: buzz.ChannelID,
		HostID:    buzz.HostID,
		EndedAt:   now,
		Status:    buzz.Status,
	}

	logger.Info("buzz %s ended by host %s", buzzID, hostID)
	return resp, http.StatusOK, nil
}

// GetBuzzMetadata returns metadata for a buzz including participants information
// Accessible to all channel members (not just buzz participants)
func GetBuzzMetadata(db *storage.Database, logger *utility.Logger, buzzID string, userID string) (models.BuzzMetadataResponse, int, error) {
	var resp models.BuzzMetadataResponse

	logger.Info("fetching metadata for buzz %s by user %s", buzzID, userID)

	// Fetch the buzz
	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Error("buzz not found: %s", buzzID)
			return resp, http.StatusNotFound, errors.New("buzz not found")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	// Verify user is a member of the channel where the buzz is active
	if !models.IsUserInChannel(db.Postgresql, buzz.ChannelID, userID) {
		logger.Error("user %s is not a member of channel %s", userID, buzz.ChannelID)
		return resp, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	// Fetch participant metadata
	participantMetadata, err := getParticipantsMetadata(db.Postgresql, buzzID)
	if err != nil {
		logger.Error("failed to fetch participant metadata: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch participant details")
	}

	resp = buildBuzzMetadataResponse(&buzz, participantMetadata)

	logger.Info("successfully fetched metadata for buzz %s with %d participants", buzzID, len(participantMetadata))
	return resp, http.StatusOK, nil
}

// GetChannelActiveBuzzIndicator returns whether a channel has an active buzz with participant preview
// Returns indicator info plus participant count and a preview of first few names
// Also checks if the requesting user is in the buzz (for member verification)
func GetChannelActiveBuzzIndicator(db *storage.Database, logger *utility.Logger, channelID, userID string) (models.ActiveBuzzIndicator, int, error) {
	var resp models.ActiveBuzzIndicator

	logger.Info("checking for active buzz in channel %s", channelID)

	// Fetch active buzz if it exists
	var buzz models.Buzz
	err := db.Postgresql.Where("channel_id = ? AND status = ? AND is_live_status = ?",
		channelID, models.BuzzStatusActive, true).First(&buzz).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No active buzz - this is normal
			logger.Info("no active buzz found in channel %s", channelID)
			resp.IsActive = false
			resp.ParticipantCount = 0
			resp.RemainingParticipants = 0
			return resp, http.StatusOK, nil
		}
		// Database error
		logger.Error("failed to check active buzz in channel %s: %v", channelID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to check active buzz")
	}

	// Active buzz found - fetch participant metadata for preview
	participantMetadata, err := getParticipantsMetadata(db.Postgresql, buzz.ID)
	if err != nil {
		logger.Error("failed to fetch participant metadata for indicator: %v", err)
		// Don't fail the entire request, just return without participant preview
		resp = models.ActiveBuzzIndicator{
			IsActive:     true,
			BuzzID:       buzz.ID,
			HostID:       buzz.HostID,
			Status:       buzz.Status,
			IsUserInBuzz: false,
		}
		return resp, http.StatusOK, nil
	}

	// Check if requesting user is in the buzz
	isUserInBuzz := false
	if userID != "" {
		for _, participant := range participantMetadata {
			if participant.UserID == userID && participant.Status == models.BuzzParticipantStatusActive {
				isUserInBuzz = true
				break
			}
		}
	}

	// Build participant preview (first 2-3 names)
	previewCount := 2
	if len(participantMetadata) < previewCount {
		previewCount = len(participantMetadata)
	}

	participantPreview := make([]string, previewCount)
	for i := 0; i < previewCount; i++ {
		participantPreview[i] = participantMetadata[i].UserName
	}

	remainingCount := len(participantMetadata) - previewCount
	if remainingCount < 0 {
		remainingCount = 0
	}

	logger.Info("active buzz found in channel %s: %s with %d participants", channelID, buzz.ID, len(participantMetadata))
	resp = models.ActiveBuzzIndicator{
		IsActive:              true,
		BuzzID:                buzz.ID,
		HostID:                buzz.HostID,
		Status:                buzz.Status,
		ParticipantCount:      len(participantMetadata),
		ParticipantPreview:    participantPreview,
		RemainingParticipants: remainingCount,
		IsUserInBuzz:          isUserInBuzz,
	}

	return resp, http.StatusOK, nil
}

// ForceEndBuzz forcefully ends an active buzz without permission checks - FOR TESTING ONLY
func ForceEndBuzz(db *storage.Database, logger *utility.Logger, buzzID string) (*models.BuzzEndResponse, int, error) {
	logger.Warning("[TEST ENDPOINT] Force ending buzz %s without permission checks", buzzID)

	// Fetch buzz directly without permission check
	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ? AND status = ?", buzzID, models.BuzzStatusActive).First(&buzz).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("buzz not found or not active: %s", buzzID)
			return nil, http.StatusNotFound, errors.New("buzz not found or not active")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	now := time.Now().UTC()

	// Update buzz status in transaction
	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		// Update buzz to ended status
		buzz.Status = models.BuzzStatusEnded
		buzz.IsLiveStatus = false
		buzz.BuzzEndTime = &now
		buzz.UpdatedAt = now

		if err := tx.Model(&buzz).Updates(buzz).Error; err != nil {
			return err
		}

		// Update all active participants to left status
		if err := tx.Model(&models.BuzzParticipant{}).
			Where("buzz_id = ? AND status = ?", buzzID, models.BuzzParticipantStatusActive).
			Updates(map[string]interface{}{
				"status":  models.BuzzParticipantStatusLeft,
				"left_at": now,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("failed to force end buzz: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to end buzz")
	}

	// Expire all pending invitations for this buzz
	if err := models.ExpireInvitationsForBuzz(db.Postgresql, buzzID); err != nil {
		logger.Error("failed to expire invitations for buzz %s: %v", buzzID, err)
	}

	// Publish buzz ended event
	eventPayload := models.BuzzEventPayload{
		Event:          string(models.BuzzEnded),
		BuzzID:         buzz.ID,
		ChannelID:      buzz.ChannelID,
		HostID:         buzz.HostID,
		ParticipantIDs: []string{},
		CreatedAt:      now,
		Status:         buzz.Status,
	}

	notification := models.Notification[models.BuzzEnded]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, buzz.ChannelID, notification); err != nil {
		logger.Error("failed to publish buzz ended event: %v", err)
	}

	logger.Info("[TEST ENDPOINT] buzz %s force ended successfully", buzzID)
	return &models.BuzzEndResponse{
		BuzzID:    buzz.ID,
		ChannelID: buzz.ChannelID,
		HostID:    buzz.HostID,
		EndedAt:   now,
		Status:    buzz.Status,
	}, http.StatusOK, nil
}
