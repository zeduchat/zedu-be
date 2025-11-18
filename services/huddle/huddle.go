package huddle

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
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

	if !models.ChannelExists(db.Postgresql, req.ChannelID) {
		return resp, http.StatusNotFound, errors.New("channel does not exist")
	}

	if !models.IsUserInChannel(db.Postgresql, req.ChannelID, hostID) {
		return resp, http.StatusForbidden, errors.New("user is not a member of the channel")
	}

	participants := uniqueParticipants(hostID, req.ParticipantIDs)

	now := time.Now().UTC()
	huddle := models.Huddle{
		ID:              uuid.NewString(),
		ChannelID:       req.ChannelID,
		HostID:          hostID,
		Participants:    participants,
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
				ID:       uuid.NewString(),
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
		HuddleID:     huddle.ID,
		HostID:       huddle.HostID,
		ChannelID:    huddle.ChannelID,
		Status:       huddle.Status,
		CreatedAt:    huddle.CreatedAt,
		StartedAt:    huddle.HuddleStartTime,
		Participants: participants,
	}

	notification := map[string]any{
		"event":        "huddle_started",
		"huddle_id":    huddle.ID,
		"channel_id":   huddle.ChannelID,
		"host_id":      huddle.HostID,
		"participants": participants,
		"created_at":   huddle.HuddleStartTime,
		"status":       huddle.Status,
	}

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
