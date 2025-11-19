package huddle

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// UpdateCameraStatus broadcasts camera status without persisting to database
func UpdateCameraStatus(db *storage.Database, logger *utility.Logger, huddleID string, req models.UpdateCameraRequest, requestingUserID string) (models.UpdateCameraResponse, int, error) {
	var resp models.UpdateCameraResponse

	if requestingUserID != req.UserID {
		return resp, http.StatusForbidden, errors.New("you can only toggle your own camera")
	}

	var huddle models.Huddle
	err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("huddle not found")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	if huddle.Status != models.HuddleStatusActive {
		return resp, http.StatusBadRequest, errors.New("huddle is not active")
	}

	var participant models.HuddleParticipant
	err = db.Postgresql.Where("huddle_id = ? AND user_id = ? AND status = ?",
		huddleID, req.UserID, models.HuddleParticipantStatusActive).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("you are not an active participant in this huddle")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	now := time.Now().UTC()
	resp = models.UpdateCameraResponse{
		HuddleID:   huddleID,
		UserID:     req.UserID,
		IsCameraOn: req.Status,
		UpdatedAt:  now.Format(time.RFC3339),
	}

	eventPayload := models.CameraStatusEventPayload{
		Event:      string(models.CameraStatusChanged),
		HuddleID:   huddleID,
		ChannelID:  huddle.ChannelID,
		UserID:     req.UserID,
		IsCameraOn: req.Status,
		Timestamp:  now.Format(time.RFC3339),
	}

	notification := models.Notification[models.CameraStatusChanged]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: huddle.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, huddle.ChannelID, notification); err != nil {
		logger.Error("failed to publish camera status event: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to broadcast camera status")
	}

	logger.Info("camera status broadcasted successfully for user %s in huddle %s", req.UserID, huddleID)
	return resp, http.StatusOK, nil
}
