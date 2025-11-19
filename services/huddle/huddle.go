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

	err = db.Postgresql.Transaction(func(tx *gorm.DB) error {
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

		if err := postgresql.CreateMultipleRecords(tx, &participantRows, len(participantRows)); err != nil {
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

	// Fetch the huddle
	var huddle models.Huddle
	if err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusBadRequest, errors.New("huddle does not exist")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	// Verify huddle is active
	if huddle.Status != models.HuddleStatusActive {
		return resp, http.StatusBadRequest, errors.New("huddle is not active")
	}

	// Check if user is in the channel
	if !models.IsUserInChannel(db.Postgresql, huddle.ChannelID, userID) {
		return resp, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	// Check if user is already in huddle
	for _, participantID := range huddle.ParticipantIDs {
		if participantID == userID {
			return resp, http.StatusBadRequest, errors.New("user is already in the huddle")
		}
	}

	timestamp := time.Now().UTC()

	// Add user to huddle
	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		// Update participants array
		if err := huddle.AddUserToHuddle(tx, userID); err != nil {
			return err
		}

		// Create participant record
		participant := models.HuddleParticipant{
			ID:       utility.GenerateUUID(),
			HuddleID: huddle.ID,
			UserID:   userID,
			Status:   models.HuddleParticipantStatusActive,
			IsMuted:  false,
			JoinedAt: timestamp,
		}

		if err := tx.Create(&participant).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("failed to add user to huddle: %v", err)
		return resp, http.StatusBadRequest, errors.New("could not update huddle participants")
	}

	// Fetch updated huddle
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

	// Emit Centrifugo event
	eventPayload := models.HuddleEventPayload{
		Event:          "user_joined_huddle",
		HuddleID:       huddle.ID,
		ChannelID:      huddle.ChannelID,
		HostID:         huddle.HostID,
		ParticipantIDs: huddle.ParticipantIDs,
		CreatedAt:      timestamp,
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
		logger.Error("failed to publish join huddle event: %v", err)
	}

	return resp, http.StatusOK, nil
}
