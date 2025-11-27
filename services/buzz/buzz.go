package buzz

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

// CreateBuzz creates a new buzz, adds the host as the first participant, and emits a realtime event.
func CreateBuzz(db *storage.Database, logger *utility.Logger, req models.CreateBuzzRequest, hostID string) (models.BuzzCreateResponse, int, error) {
	var resp models.BuzzCreateResponse

	chModel := models.Channels{}
	exists, err := chModel.CheckChannelExists(db.Postgresql, req.ChannelID)
	if err != nil {
		logger.Error("failed to verify channel existence: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify channel")
	}
	if !exists {
		return resp, http.StatusNotFound, errors.New("channel does not exist")
	}

	if !models.IsUserInChannel(db.Postgresql, req.ChannelID, hostID) {
		return resp, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	participants := uniqueParticipants(hostID, req.ParticipantIDs)

	now := time.Now().UTC()
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

		participantRows := make([]models.BuzzParticipant, 0, len(participants))
		for _, pid := range participants {
			participantRows = append(participantRows, models.BuzzParticipant{
				ID:       utility.GenerateUUID(),
				BuzzID:   buzz.ID,
				UserID:   pid,
				Status:   models.BuzzParticipantStatusActive,
				IsMuted:  false,
				JoinedAt: now,
			})
		}

		if err := postgresql.CreateMultipleRecords(tx, &participantRows, len(participantRows)); err != nil {
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

func uniqueParticipants(hostID string, extras []string) []string {
	seen := map[string]bool{}
	var result []string

	seen[hostID] = true
	result = append(result, hostID)

	for _, id := range extras {
		if id == "" {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}

	return result
}

// Joinbuzz allows a user to join an existing buzz
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
	var buzz models.Buzz

	if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return buzz, http.StatusBadRequest, errors.New("buzz does not exist")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return buzz, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	if buzz.Status != models.BuzzStatusActive {
		return buzz, http.StatusBadRequest, errors.New("buzz is not active")
	}

	if !models.IsUserInChannel(db.Postgresql, buzz.ChannelID, userID) {
		return buzz, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	for _, participantID := range buzz.ParticipantIDs {
		if participantID == userID {
			return buzz, http.StatusBadRequest, errors.New("user is already in the buzz")
		}
	}

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
