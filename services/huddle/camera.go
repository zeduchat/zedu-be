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

// UpdateCameraStatus updates the camera status for a participant in a huddle
func UpdateCameraStatus(db *storage.Database, logger *utility.Logger, huddleID string, req models.UpdateCameraRequest, requestingUserID string) (models.UpdateCameraResponse, int, error) {
	var resp models.UpdateCameraResponse

	// Verify the requesting user is the same as the user in the request
	if requestingUserID != req.UserID {
		return resp, http.StatusForbidden, errors.New("you can only toggle your own camera")
	}

	// Check if huddle exists and is active
	var huddle models.Huddle
	err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("huddle not found")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	// Check if huddle is still active
	if huddle.Status != models.HuddleStatusActive {
		return resp, http.StatusBadRequest, errors.New("huddle is not active")
	}

	// Find the participant record
	var participant models.HuddleParticipant
	err = db.Postgresql.Where("huddle_id = ? AND user_id = ?", huddleID, req.UserID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("you are not a participant in this huddle")
		}
		logger.Error("failed to fetch participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch participant")
	}

	// Check if participant is still active
	if participant.Status != models.HuddleParticipantStatusActive {
		return resp, http.StatusBadRequest, errors.New("you are no longer active in this huddle")
	}

	// Update camera status
	now := time.Now().UTC()
	err = db.Postgresql.Model(&participant).Updates(map[string]interface{}{
		"is_camera_on": req.Status,
	}).Error
	if err != nil {
		logger.Error("failed to update camera status: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to update camera status")
	}

	// Build response
	resp = models.UpdateCameraResponse{
		HuddleID:   huddleID,
		UserID:     req.UserID,
		IsCameraOn: req.Status,
		UpdatedAt:  now.Format(time.RFC3339),
	}

	// Emit real-time event
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
		// Don't fail the request if event emission fails
	}

	logger.Info("camera status updated successfully for user %s in huddle %s", req.UserID, huddleID)
	return resp, http.StatusOK, nil
}
