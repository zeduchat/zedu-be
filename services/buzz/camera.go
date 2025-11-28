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

// UpdateCameraStatus broadcasts camera status
func UpdateCameraStatus(db *storage.Database, logger *utility.Logger, buzzID string, req models.UpdateCameraRequest, requestingUserID string) (models.UpdateCameraResponse, int, error) {
	var resp models.UpdateCameraResponse

	if requestingUserID != req.UserID {
		return resp, http.StatusForbidden, errors.New("you can only toggle your own camera")
	}

	// Validate buzz is active and user is an active participant (buzz state + participant validation)
	buzz, err := permissions.CanPerformBuzzAction(db.Postgresql, buzzID, requestingUserID)
	if err != nil {
		if err == permissions.ErrBuzzNotFound {
			return resp, http.StatusNotFound, errors.New("buzz not found")
		}
		if err == permissions.ErrBuzzEnded {
			return resp, http.StatusConflict, errors.New("buzz has ended")
		}
		if err == permissions.ErrNotActiveParticipant {
			return resp, http.StatusForbidden, errors.New("you are not an active participant in this buzz")
		}
		logger.Error("failed to validate permissions: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to validate permissions")
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
