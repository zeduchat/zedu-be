package devicetoken

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

// RegisterDeviceToken saves or updates a device token for a user, allowing multiple devices per user.
func RegisterDeviceToken(db *storage.Database, logger *utility.Logger, req models.RegisterDeviceTokenRequest, userID string) (models.RegisterDeviceTokenResponse, int, error) {
	var resp models.RegisterDeviceTokenResponse
	now := time.Now().UTC()

	var token models.UserDeviceToken
	err := db.Postgresql.Where("user_id = ? AND device_token = ?", userID, req.DeviceToken).First(&token).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("failed to query device token: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch device token")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		token = models.UserDeviceToken{
			ID:          utility.GenerateUUID(),
			UserID:      userID,
			DeviceToken: req.DeviceToken,
			Platform:    req.Platform,
			DeviceID:    req.DeviceID,
			LastSeen:    now,
		}

		if err := db.Postgresql.Create(&token).Error; err != nil {
			logger.Error("failed to create device token: %v", err)
			return resp, http.StatusInternalServerError, errors.New("failed to save device token")
		}

		return toDeviceTokenResponse(token), http.StatusCreated, nil
	}

	updates := map[string]any{
		"last_seen": now,
	}
	if req.Platform != "" {
		updates["platform"] = req.Platform
	}
	if req.DeviceID != "" {
		updates["device_id"] = req.DeviceID
	}

	if err := db.Postgresql.Model(&token).Updates(updates).Error; err != nil {
		logger.Error("failed to update device token: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to update device token")
	}

	if v, ok := updates["platform"]; ok {
		token.Platform = v.(string)
	}
	if v, ok := updates["device_id"]; ok {
		token.DeviceID = v.(string)
	}
	token.LastSeen = now

	return toDeviceTokenResponse(token), http.StatusOK, nil
}

func toDeviceTokenResponse(token models.UserDeviceToken) models.RegisterDeviceTokenResponse {
	return models.RegisterDeviceTokenResponse{
		ID:          token.ID,
		UserID:      token.UserID,
		DeviceToken: token.DeviceToken,
		Platform:    token.Platform,
		DeviceID:    token.DeviceID,
		LastSeen:    token.LastSeen,
		UpdatedAt:   token.UpdatedAt,
	}
}
