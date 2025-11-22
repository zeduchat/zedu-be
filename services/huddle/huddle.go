package huddle

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

// CreateHuddle creates a new huddle, adds the host as the first participant, and emits a realtime event.
func CreateHuddle(db *storage.Database, logger *utility.Logger, req models.CreateHuddleRequest, hostID string) (models.HuddleCreateResponse, int, error) {
	var resp models.HuddleCreateResponse

	var channel models.Channels
	if err := db.Postgresql.Where("id = ?", req.ChannelID).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("channel does not exist")
		}
		logger.Error("failed to fetch channel: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch channel")
	}

	if !models.IsUserInChannel(db.Postgresql, req.ChannelID, hostID) {
		return resp, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	participants := uniqueParticipants(hostID, req.ParticipantIDs)

	now := time.Now().UTC()
	huddle := models.Huddle{
		ID:              utility.GenerateUUID(),
		ChannelID:       req.ChannelID,
		HostID:          hostID,
		ParticipantIDs:  participants,
		HuddleStartTime: now,
		IsLiveStatus:    true,
		Status:          models.HuddleStatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&huddle).Error; err != nil {
			return err
		}

		participantRows := make([]models.HuddleParticipant, 0, len(participants))
		for _, pid := range participants {
			participantRows = append(participantRows, models.HuddleParticipant{
				ID:       utility.GenerateUUID(),
				HuddleID: huddle.ID,
				UserID:   pid,
				Status:   models.HuddleParticipantStatusActive,
				IsMuted:  false,
				JoinedAt: now,
			})
		}

		if err := postgresql.CreateMultipleRecords(tx, &participantRows, len(participants)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.Error("failed to create huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to create huddle")
	}

	resp = models.HuddleCreateResponse{
		HuddleID:       huddle.ID,
		HostID:         huddle.HostID,
		ChannelID:      huddle.ChannelID,
		Status:         huddle.Status,
		CreatedAt:      huddle.CreatedAt,
		StartedAt:      huddle.HuddleStartTime,
		ParticipantIDs: participants,
	}

	eventPayload := models.HuddleEventPayload{
		Event:          string(models.HuddleStarted),
		HuddleID:       huddle.ID,
		ChannelID:      huddle.ChannelID,
		HostID:         huddle.HostID,
		ParticipantIDs: participants,
		CreatedAt:      huddle.HuddleStartTime,
		Status:         huddle.Status,
	}

	notification := models.Notification[models.HuddleStarted]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: huddle.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, huddle.ChannelID, notification); err != nil {
		logger.Error("failed to publish huddle event: %v", err)
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

// JoinHuddle allows a user to join an existing huddle
func JoinHuddle(db *storage.Database, logger *utility.Logger, huddleID string, userID string) (models.JoinHuddleResponse, int, error) {
	var resp models.JoinHuddleResponse

	huddle, statusCode, err := validateJoinHuddle(db, logger, huddleID, userID)
	if err != nil {
		return resp, statusCode, err
	}

	timestamp := time.Now().UTC()

	if err := addUserToHuddleTransaction(db, logger, &huddle, userID, timestamp); err != nil {
		return resp, http.StatusBadRequest, err
	}

	if err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error; err != nil {
		logger.Error("failed to fetch updated huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch updated huddle")
	}

	resp = models.JoinHuddleResponse{
		HuddleID:       huddle.ID,
		ChannelID:      huddle.ChannelID,
		UserID:         userID,
		Status:         huddle.Status,
		JoinedAt:       timestamp,
		ParticipantIDs: huddle.ParticipantIDs,
	}

	publishJoinHuddleEvent(logger, huddle, timestamp)

	return resp, http.StatusOK, nil
}

func validateJoinHuddle(db *storage.Database, logger *utility.Logger, huddleID, userID string) (models.Huddle, int, error) {
	var huddle models.Huddle

	if err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return huddle, http.StatusBadRequest, errors.New("huddle does not exist")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return huddle, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	if huddle.Status != models.HuddleStatusActive {
		return huddle, http.StatusBadRequest, errors.New("huddle is not active")
	}

	if !models.IsUserInChannel(db.Postgresql, huddle.ChannelID, userID) {
		return huddle, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	for _, participantID := range huddle.ParticipantIDs {
		if participantID == userID {
			return huddle, http.StatusBadRequest, errors.New("user is already in the huddle")
		}
	}

	return huddle, http.StatusOK, nil
}

func addUserToHuddleTransaction(db *storage.Database, logger *utility.Logger, huddle *models.Huddle, userID string, timestamp time.Time) error {
	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := huddle.AddUserToHuddle(tx, userID); err != nil {
			return err
		}

		participant := models.HuddleParticipant{
			ID:       utility.GenerateUUID(),
			HuddleID: huddle.ID,
			UserID:   userID,
			Status:   models.HuddleParticipantStatusActive,
			IsMuted:  false,
			JoinedAt: timestamp,
		}

		return tx.Create(&participant).Error
	})

	if err != nil {
		logger.Error("failed to add user to huddle: %v", err)
		return errors.New("could not update huddle participants")
	}

	return nil
}

func publishJoinHuddleEvent(logger *utility.Logger, huddle models.Huddle, timestamp time.Time) {
	eventPayload := models.HuddleEventPayload{
		Event:          string(models.UserJoinedHuddle),
		HuddleID:       huddle.ID,
		ChannelID:      huddle.ChannelID,
		HostID:         huddle.HostID,
		ParticipantIDs: huddle.ParticipantIDs,
		CreatedAt:      timestamp,
		Status:         huddle.Status,
	}

	notification := models.Notification[models.UserJoinedHuddle]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: huddle.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, huddle.ChannelID, notification); err != nil {
		logger.Error("failed to publish join huddle event: %v", err)
	}
}
