package mediaPreferences

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

// ValidateDeviceExists checks if a device_id exists in the device registry
func ValidateDeviceExists(db *gorm.DB, deviceID string) error {
	if deviceID == "" {
		return nil
	}

	// Validate UUID format
	_, err := uuid.FromString(deviceID)
	if err != nil {
		return fmt.Errorf("invalid device_id format")
	}

	// Check if device exists in device registry
	var device models.DeviceRegistry
	exists := postgresql.CheckExists(db, &device, "id = ?", deviceID)
	if !exists {
		return fmt.Errorf("device not found")
	}

	return nil
}

// UpdateMediaPreferences updates media preferences for a user, with optional device-specific overrides
func UpdateMediaPreferences(req models.UpdateMediaPreferencesRequest, userID string, db *gorm.DB, logger *utility.Logger) (*models.MediaPreferencesResponse, int, error) {
	var pref models.MediaPreferences

	// Validate user exists
	var user models.User
	exists := postgresql.CheckExists(db, &user, "id = ?", userID)
	if !exists {
		return nil, http.StatusNotFound, errors.New("user not found")
	}

	// Validate device_id if provided
	if req.DeviceID != nil && *req.DeviceID != "" {
		if err := ValidateDeviceExists(db, *req.DeviceID); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("device not found: %v", err)
		}
	}

	// Validate individual field values
	if req.AutoDownloadPhotos != "" {
		if err := models.ValidateAutoDownload(req.AutoDownloadPhotos); err != nil {
			return nil, http.StatusBadRequest, err
		}
	}
	if req.AutoDownloadAudio != "" {
		if err := models.ValidateAutoDownload(req.AutoDownloadAudio); err != nil {
			return nil, http.StatusBadRequest, err
		}
	}
	if req.AutoDownloadDocuments != "" {
		if err := models.ValidateAutoDownload(req.AutoDownloadDocuments); err != nil {
			return nil, http.StatusBadRequest, err
		}
	}
	if req.AutoDownloadVideos != "" {
		if err := models.ValidateAutoDownload(req.AutoDownloadVideos); err != nil {
			return nil, http.StatusBadRequest, err
		}
	}
	if req.UploadQuality != "" {
		if err := models.ValidateUploadQuality(req.UploadQuality); err != nil {
			return nil, http.StatusBadRequest, err
		}
	}

	// Determine if this is a device-specific or user-level update
	if req.DeviceID != nil && *req.DeviceID != "" {
		// Device-specific preferences
		deviceUUID, err := uuid.FromString(*req.DeviceID)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid device_id format")
		}

		// Check if device preferences exist
		err = db.Where("user_id = ? AND device_id = ?", userID, deviceUUID).First(&pref).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new device preferences
			userUUID, err := uuid.FromString(userID)
			if err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("invalid user_id format")
			}

			// Get user-level preferences for defaults
			userPref, err := pref.GetOrCreateUserPreferences(db, userID)
			if err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to get user preferences: %v", err)
			}

			pref = models.MediaPreferences{
				ID:                    utility.GenerateUUID(),
				UserID:                userUUID,
				DeviceID:              &deviceUUID,
				AutoDownloadPhotos:    userPref.AutoDownloadPhotos,
				AutoDownloadAudio:     userPref.AutoDownloadAudio,
				AutoDownloadDocuments: userPref.AutoDownloadDocuments,
				AutoDownloadVideos:    userPref.AutoDownloadVideos,
				UploadQuality:         userPref.UploadQuality,
			}

			// Update only provided fields
			updates := make(map[string]interface{})
			if req.AutoDownloadPhotos != "" {
				updates["auto_download_photos"] = req.AutoDownloadPhotos
				pref.AutoDownloadPhotos = req.AutoDownloadPhotos
			}
			if req.AutoDownloadAudio != "" {
				updates["auto_download_audio"] = req.AutoDownloadAudio
				pref.AutoDownloadAudio = req.AutoDownloadAudio
			}
			if req.AutoDownloadDocuments != "" {
				updates["auto_download_documents"] = req.AutoDownloadDocuments
				pref.AutoDownloadDocuments = req.AutoDownloadDocuments
			}
			if req.AutoDownloadVideos != "" {
				updates["auto_download_videos"] = req.AutoDownloadVideos
				pref.AutoDownloadVideos = req.AutoDownloadVideos
			}
			if req.UploadQuality != "" {
				updates["upload_quality"] = req.UploadQuality
				pref.UploadQuality = req.UploadQuality
			}

			if err := pref.Create(db); err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to create device preferences: %v", err)
			}
		} else if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to get device preferences: %v", err)
		} else {
			// Update existing device preferences (partial update)
			updates := make(map[string]interface{})
			if req.AutoDownloadPhotos != "" {
				updates["auto_download_photos"] = req.AutoDownloadPhotos
			}
			if req.AutoDownloadAudio != "" {
				updates["auto_download_audio"] = req.AutoDownloadAudio
			}
			if req.AutoDownloadDocuments != "" {
				updates["auto_download_documents"] = req.AutoDownloadDocuments
			}
			if req.AutoDownloadVideos != "" {
				updates["auto_download_videos"] = req.AutoDownloadVideos
			}
			if req.UploadQuality != "" {
				updates["upload_quality"] = req.UploadQuality
			}

			if len(updates) > 0 {
				if err := pref.Update(db, updates); err != nil {
					return nil, http.StatusInternalServerError, fmt.Errorf("failed to update device preferences: %v", err)
				}

				// Reload to get updated values
				if err := db.Where("user_id = ? AND device_id = ?", userID, deviceUUID).First(&pref).Error; err != nil {
					return nil, http.StatusInternalServerError, fmt.Errorf("failed to reload device preferences: %v", err)
				}
			}
		}
	} else {
		// User-level preferences
		userPref, err := pref.GetOrCreateUserPreferences(db, userID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to get user preferences: %v", err)
		}

		pref = userPref

		// Update only provided fields
		updates := make(map[string]interface{})
		if req.AutoDownloadPhotos != "" {
			updates["auto_download_photos"] = req.AutoDownloadPhotos
		}
		if req.AutoDownloadAudio != "" {
			updates["auto_download_audio"] = req.AutoDownloadAudio
		}
		if req.AutoDownloadDocuments != "" {
			updates["auto_download_documents"] = req.AutoDownloadDocuments
		}
		if req.AutoDownloadVideos != "" {
			updates["auto_download_videos"] = req.AutoDownloadVideos
		}
		if req.UploadQuality != "" {
			updates["upload_quality"] = req.UploadQuality
		}

		if len(updates) > 0 {
			if err := pref.Update(db, updates); err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to update user preferences: %v", err)
			}

			// Reload to get updated values
			if err := db.Where("user_id = ? AND device_id IS NULL", userID).First(&pref).Error; err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to reload user preferences: %v", err)
			}
		}
	}

	response := pref.ToResponse()
	return &response, http.StatusOK, nil
}

// GetMediaPreferences retrieves media preferences for a user, with optional device-specific overrides
func GetMediaPreferences(userID string, deviceID *string, db *gorm.DB, logger *utility.Logger) (*models.MediaPreferencesResponse, int, error) {
	var pref models.MediaPreferences

	// Validate user exists
	var user models.User
	exists := postgresql.CheckExists(db, &user, "id = ?", userID)
	if !exists {
		return nil, http.StatusNotFound, errors.New("user not found")
	}

	// Validate device_id if provided
	if deviceID != nil && *deviceID != "" {
		if err := ValidateDeviceExists(db, *deviceID); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("device not found: %v", err)
		}
	}

	// Get preferences with fallback logic
	result, err := pref.GetDevicePreferences(db, userID, deviceID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No preferences exist, create default user-level preferences
		userPref, err := pref.GetOrCreateUserPreferences(db, userID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to create default preferences: %v", err)
		}
		response := userPref.ToResponse()
		return &response, http.StatusOK, nil
	}

	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to get preferences: %v", err)
	}

	response := result.ToResponse()
	return &response, http.StatusOK, nil
}

// ResetAutoDownloadSettings resets all auto-download settings to default values ("wifi_only")
// for a user, with optional device-specific reset
func ResetAutoDownloadSettings(userID string, deviceID *string, db *gorm.DB, logger *utility.Logger) (*models.MediaPreferencesResponse, int, error) {
	var pref models.MediaPreferences
	
	// Validate user exists
	var user models.User
	exists := postgresql.CheckExists(db, &user, "id = ?", userID)
	if !exists {
		return nil, http.StatusNotFound, errors.New("user not found")
	}
	
	// Validate device_id if provided
	if deviceID != nil && *deviceID != "" {
		if err := ValidateDeviceExists(db, *deviceID); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("device not found: %v", err)
		}
	}
	
	// Determine if this is a device-specific or user-level reset
	if deviceID != nil && *deviceID != "" {
		// Device-specific reset
		deviceUUID, err := uuid.FromString(*deviceID)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("invalid device_id format")
		}
		
		// Check if device preferences exist
		err = db.Where("user_id = ? AND device_id = ?", userID, deviceUUID).First(&pref).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new device preferences with defaults
			userUUID, err := uuid.FromString(userID)
			if err != nil {
				return nil, http.StatusBadRequest, fmt.Errorf("invalid user_id format")
			}
			
			// Get user-level preferences for defaults
			userPref, err := pref.GetOrCreateUserPreferences(db, userID)
			if err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to get user preferences: %v", err)
			}
			
			pref = models.MediaPreferences{
				ID:                    utility.GenerateUUID(),
				UserID:                userUUID,
				DeviceID:              &deviceUUID,
				AutoDownloadPhotos:    "wifi_only",
				AutoDownloadAudio:     "wifi_only",
				AutoDownloadDocuments: "wifi_only",
				AutoDownloadVideos:    "wifi_only",
				UploadQuality:         userPref.UploadQuality, // Keep user-level upload quality
			}
			
			if err := pref.Create(db); err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to create device preferences: %v", err)
			}
		} else if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to get device preferences: %v", err)
		} else {
			// Reset existing device preferences
			updates := map[string]interface{}{
				"auto_download_photos":    "wifi_only",
				"auto_download_audio":     "wifi_only",
				"auto_download_documents": "wifi_only",
				"auto_download_videos":    "wifi_only",
			}
			
			if err := pref.Update(db, updates); err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to reset device preferences: %v", err)
			}
			
			// Reload to get updated values
			if err := db.Where("user_id = ? AND device_id = ?", userID, deviceUUID).First(&pref).Error; err != nil {
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to reload device preferences: %v", err)
			}
		}
	} else {
		// User-level reset
		userPref, err := pref.GetOrCreateUserPreferences(db, userID)
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to get user preferences: %v", err)
		}
		
		pref = userPref
		
		// Reset user-level preferences
		updates := map[string]interface{}{
			"auto_download_photos":    "wifi_only",
			"auto_download_audio":     "wifi_only",
			"auto_download_documents": "wifi_only",
			"auto_download_videos":    "wifi_only",
		}
		
		if err := pref.Update(db, updates); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to reset user preferences: %v", err)
		}
		
		// Reload to get updated values
		if err := db.Where("user_id = ? AND device_id IS NULL", userID).First(&pref).Error; err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to reload user preferences: %v", err)
		}
	}
	
	response := pref.ToResponse()
	return &response, http.StatusOK, nil
}
