package buzz

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
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

// CreateBuzz creates a new buzz, adds the host as the first participant, and emits a realtime event.
func CreateBuzz(db *storage.Database, logger *utility.Logger, req models.CreateBuzzRequest, hostID string) (models.BuzzCreateResponse, int, error) {
	var resp models.BuzzCreateResponse

	// Validate permissions using centralized permission check
	err := permissions.CanCreateBuzz(db.Postgresql, req.ChannelID, hostID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "create")
		logger.Error("permission check failed for user %s creating buzz in channel %s: %v", hostID, req.ChannelID, err)
		return resp, statusCode, errors.New(errMsg)
	}

	// Determine channel type (regular, DM, or group DM)
	channelType, err := permissions.GetChannelType(db.Postgresql, req.ChannelID)
	if err != nil {
		logger.Error("failed to determine channel type for channel %s: %v", req.ChannelID, err)
		return resp, http.StatusInternalServerError, errors.New("failed to determine channel type")
	}

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
		ChannelID:      req.ChannelID,
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
	token, err := service.GenerateRTCToken(buzz.ID, hostID, hostID, agora.DefaultTokenExpirationSeconds)
	if err != nil {
		logger.Error("failed to generate Agora token for host: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	agoraToken := models.BuzzAgoraTokenResponse{
		Token:       token,
		AppId:       service.GetAppId(),
		ChannelName: buzz.ID,
		UID:         hostID,
	}

	err = db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&buzz).Error; err != nil {
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
			return err
		}

		return nil
	})
	if err != nil {
		logger.Error("failed to create buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to create buzz")
	}

	// Fetch participant metadata
	participantMetadata, err := getParticipantsMetadata(db.Postgresql, buzz.ID)
	if err != nil {
		logger.Error("failed to fetch participant metadata: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch participant details")
	}

	resp = models.BuzzCreateResponse{
		BuzzID:       buzz.ID,
		HostID:       buzz.HostID,
		ChannelID:    buzz.ChannelID,
		Status:       buzz.Status,
		CreatedAt:    buzz.CreatedAt,
		StartedAt:    buzz.BuzzStartTime,
		Participants: participantMetadata,
		AgoraToken:   &agoraToken,
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

	return resp, http.StatusCreated, nil
}

// JoinBuzz allows a user to join an existing buzz
func JoinBuzz(db *storage.Database, logger *utility.Logger, buzzID string, userID string) (models.JoinBuzzResponse, int, error) {
	var resp models.JoinBuzzResponse

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
		logger.Error("permission check failed for user %s joining buzz %s: %v", userID, buzzID, err)
		return resp, statusCode, errors.New(errMsg)
	}

	timestamp := time.Now().UTC()

	// Generate Agora token BEFORE adding user to buzz (using userID as UID)
	// This way if token generation fails, we haven't polluted the database
	// Use constant for token expiration (4 hours)
	token, err := service.GenerateRTCToken(buzzID, userID, userID, agora.DefaultTokenExpirationSeconds)
	if err != nil {
		logger.Error("failed to generate Agora token for user: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	agoraToken := models.BuzzAgoraTokenResponse{
		Token:       token,
		AppId:       service.GetAppId(),
		ChannelName: buzzID,
		UID:         userID,
	}

	// Add user to buzz in transaction
	if err := addUserToBuzzTransaction(db, logger, buzz, userID, timestamp); err != nil {
		logger.Error("failed to add user to buzz transaction: %v", err)
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

	resp = models.JoinBuzzResponse{
		BuzzID:       buzz.ID,
		HostID:       buzz.HostID,
		ChannelID:    buzz.ChannelID,
		UserID:       userID,
		Status:       buzz.Status,
		JoinedAt:     timestamp,
		Participants: participantMetadata,
		AgoraToken:   &agoraToken,
	}

	publishJoinBuzzEvent(logger, *buzz, timestamp)

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

		participant := models.BuzzParticipant{
			ID:       utility.GenerateUUID(),
			BuzzID:   buzz.ID,
			UserID:   userID,
			Status:   models.BuzzParticipantStatusActive,
			IsMuted:  false,
			JoinedAt: timestamp,
		}

		return tx.Create(&participant).Error
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

	buzz, status, err := validateLeaveBuzz(db, logger, buzzID, userID)
	if err != nil {
		return nil, status, err
	}

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
	logger.Info(buzz.Status)

	if buzzEnded {
		if err := models.ExpireInvitationsForBuzz(db.Postgresql, buzzID); err != nil {
			logger.Error("failed to expire invitations for buzz %s: %v", buzzID, err)
		}
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
	// Use centralized permission check
	buzz, err := permissions.CanPerformHostAction(db.Postgresql, buzzID, hostID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "end")
		logger.Error("permission check failed for user %s ending buzz %s: %v", hostID, buzzID, err)
		return nil, statusCode, errors.New(errMsg)
	}

	now := time.Now().UTC()

	// Update buzz status in transaction
	err = db.Postgresql.Transaction(func(tx *gorm.DB) error {
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
