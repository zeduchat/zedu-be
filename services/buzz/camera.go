package buzz

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

// UpdateCameraStatus broadcasts camera status
func UpdateCameraStatus(db *storage.Database, logger *utility.Logger, buzzID string, req models.UpdateCameraRequest, requestingUserID string) (models.UpdateCameraResponse, int, error) {
	var resp models.UpdateCameraResponse

	if requestingUserID != req.UserID {
		return resp, http.StatusForbidden, errors.New("you can only toggle your own camera")
	}

	var buzz models.Buzz
	err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("buzz not found")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	if buzz.Status != models.BuzzStatusActive {
		return resp, http.StatusBadRequest, errors.New("buzz is not active")
	}

	var participant models.BuzzParticipant
	err = db.Postgresql.Where("buzz_id = ? AND user_id = ? AND status = ?",
		buzzID, req.UserID, models.BuzzParticipantStatusActive).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("you are not an active participant in this buzz")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	now := time.Now().UTC()
	resp = models.UpdateCameraResponse{
		BuzzID:     buzzID,
		UserID:     req.UserID,
		IsCameraOn: req.Status,
		UpdatedAt:  now.Format(time.RFC3339),
	}

	eventPayload := models.CameraStatusEventPayload{
		Event:      string(models.CameraStatusChanged),
		BuzzID:     buzzID,
		ChannelID:  buzz.ChannelID,
		UserID:     req.UserID,
		IsCameraOn: req.Status,
		Timestamp:  now.Format(time.RFC3339),
	}

	notification := models.Notification[models.CameraStatusChanged]
	notification.SectionType = models.ChannelsSection
	notification.Content = eventPayload
	notification.ModificationDetails = &models.ModificationDetails{
		ChannelId: buzz.ChannelID,
	}
	notification.NotificationId = utility.GenerateUUID()

	if err := centrifuge.PublishChannel(logger, buzz.ChannelID, notification); err != nil {
		logger.Error("failed to publish camera status event: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to broadcast camera status")
	}

	logger.Info("camera status broadcasted successfully for user %s in buzz %s", req.UserID, buzzID)
	return resp, http.StatusOK, nil
}
