package buzz

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// CreateBuzz creates a new buzz, adds the host as the first participant, and emits a realtime event.
func CreateBuzz(db *storage.Database, logger *utility.Logger, req models.CreateBuzzRequest, hostID string) (models.BuzzCreateResponse, int, error) {
	var resp models.BuzzCreateResponse

	// Validate permissions (channel existence, host membership, concurrent buzz prevention)
	err := permissions.CanCreateBuzz(db.Postgresql, req.ChannelID, hostID)
	if err != nil {
		if err == permissions.ErrChannelNotFound {
			return resp, http.StatusNotFound, errors.New("channel does not exist")
		}
		if err == permissions.ErrNotChannelMember {
			return resp, http.StatusForbidden, errors.New("user is not a member of the channel")
		}
		if err == permissions.ErrBuzzAlreadyActive {
			return resp, http.StatusConflict, errors.New("channel already has an active buzz")
		}
		logger.Error("permission validation failed: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to validate permissions")
	}

	now := time.Now().UTC()
	// Only the host auto-joins on creation; others must explicitly join via /join endpoint
	participants := []string{hostID}

	buzz := models.Buzz{
		ID:             utility.GenerateUUID(),
		ChannelID:      req.ChannelID,
		HostID:         hostID,
		ParticipantIDs: participants,
		BuzzStartTime:  now,
		IsLiveStatus:   true,
		Status:         models.BuzzStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
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

	resp = models.BuzzCreateResponse{
		BuzzID:         buzz.ID,
		HostID:         buzz.HostID,
		ChannelID:      buzz.ChannelID,
		Status:         buzz.Status,
		CreatedAt:      buzz.CreatedAt,
		StartedAt:      buzz.BuzzStartTime,
		ParticipantIDs: participants,
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

	buzz, statusCode, err := validateJoinBuzz(db, logger, buzzID, userID)
	if err != nil {
		return resp, statusCode, err
	}

	timestamp := time.Now().UTC()

	if err := addUserToBuzzTransaction(db, logger, &buzz, userID, timestamp); err != nil {
		return resp, http.StatusBadRequest, err
	}

	if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		logger.Error("failed to fetch updated buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch updated buzz")
	}

	resp = models.JoinBuzzResponse{
		BuzzID:         buzz.ID,
		ChannelID:      buzz.ChannelID,
		UserID:         userID,
		Status:         buzz.Status,
		JoinedAt:       timestamp,
		ParticipantIDs: buzz.ParticipantIDs,
	}

	publishJoinBuzzEvent(logger, buzz, timestamp)

	return resp, http.StatusOK, nil
}

func validateJoinBuzz(db *storage.Database, logger *utility.Logger, buzzID, userID string) (models.Buzz, int, error) {
	buzz, status, err := validateBuzz(db, logger, buzzID, userID)

	if err != nil {
		logger.Error("failed to validate buzz: %v", err)
		return buzz, status, err
	}

	return buzz, http.StatusOK, nil

}

func validateLeaveBuzz(db *storage.Database, logger *utility.Logger, buzzID, userID string) (models.Buzz, int, error) {
	buzz, status, err := validateBuzz(db, logger, buzzID, userID)

	if err != nil {
		logger.Error("failed to validate buzz: %v", err)
		return buzz, status, err
	}

	if buzz.Status != models.BuzzStatusActive {
		return buzz, http.StatusBadRequest, fmt.Errorf("call has ended.")
	}

	seenUser := false
	for _, participantID := range buzz.ParticipantIDs {
		if participantID == userID {
			seenUser = true
			break
		}
	}
	if !(seenUser) {
		return buzz, http.StatusBadRequest, errors.New("user is not in the buzz")
	}

	return buzz, http.StatusOK, nil
}

func validateBuzz(db *storage.Database, logger *utility.Logger, buzzID, userID string) (models.Buzz, int, error) {
	var buzz models.Buzz

	// Validate permissions (buzz state, channel membership, not already in buzz)
	activeBuzz, err := permissions.CanJoinBuzz(db.Postgresql, buzzID, userID)
	if err != nil {
		if err == permissions.ErrBuzzNotFound {
			return buzz, http.StatusNotFound, errors.New("buzz does not exist")
		}
		if err == permissions.ErrBuzzEnded {
			return buzz, http.StatusConflict, errors.New("buzz has ended")
		}
		if err == permissions.ErrNotChannelMember {
			return buzz, http.StatusForbidden, errors.New("user is not a member of the channel")
		}
		if err == permissions.ErrAlreadyInBuzz {
			return buzz, http.StatusBadRequest, errors.New("user is already in the buzz")
		}
		logger.Error("failed to validate join permissions: %v", err)
		return buzz, http.StatusInternalServerError, errors.New("failed to validate permissions")
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

	return &models.BuzzLeaveResponse{
		BuzzID:        buzzID,
		ParticipantID: userID,
		NewHostID:     newHostID,
		LeftAt:        time.Now().UTC(),
		BuzzEnded:     buzzEnded,
	}, http.StatusOK, nil
}
