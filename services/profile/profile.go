package profile

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func GetUserProfile(db *gorm.DB, userID string) (*models.ProfileSummary, int, error) {
	var user models.User
	userProfile, err := user.GetUserWithProfile(db, userID)

	if err != nil {
		return nil, http.StatusNotFound, err
	}

	profileSummary := constructProfileSummary(userProfile)

	return profileSummary, http.StatusOK, nil
}

func IsSameOrganization(db *gorm.DB, reqUserID string, targetUserID string) (int, error) {
	var user models.User
	var org models.Organisation

	userProfile, err := user.GetUserByID(db, reqUserID)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	currentOrgID := (userProfile.CurrentOrg).String()

	isMember, err := org.CheckUserIsMemberOfOrg(targetUserID, currentOrgID, db)
	if err != nil {
		return http.StatusBadRequest, err
	}
	if !isMember {
		return http.StatusBadRequest, errors.New("user not authorised to retrieve this organisation")
	}

	return http.StatusOK, nil
}

// VerifyAvatarOwnership verifies that the authenticated user owns the target profile
// Returns true if ownership is verified, false otherwise
func VerifyAvatarOwnership(db *gorm.DB, authenticatedUserID, targetUserID string) (bool, error) {
	// Verify that authenticated user ID matches target user ID
	if authenticatedUserID != targetUserID {
		return false, nil
	}

	// Verify that the target user exists
	var user models.User
	_, err := postgresql.SelectOneFromDb(db, &user, "id = ?", targetUserID)
	if err != nil {
		return false, fmt.Errorf("user not found: %w", err)
	}

	return true, nil
}

// logUnauthorizedAvatarAccess logs unauthorized avatar access attempts
func logUnauthorizedAvatarAccess(logger *utility.Logger, authenticatedUserID, targetUserID, operation string) {
	logger.Error(
		"Unauthorized avatar access attempt",
		"authenticated_user_id", authenticatedUserID,
		"target_user_id", targetUserID,
		"operation", operation,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

func UpdateUserProfile(req models.UpdateUserProfileRequest, db *gorm.DB, logger *utility.Logger, userId string, ext string, file []byte) (int, *models.Profile, error) {
	var user models.User
	var userProfile models.Profile

	// Verify ownership before allowing any profile updates (including avatar)
	// userId is the authenticated user ID from JWT, so we verify it matches and user exists
	isOwner, err := VerifyAvatarOwnership(db, userId, userId)
	if err != nil {
		logger.Error("Failed to verify avatar ownership", "error", err, "user_id", userId)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if !isOwner {
		logUnauthorizedAvatarAccess(logger, userId, userId, "update")
		return http.StatusForbidden, nil, errors.New("you do not have permission to modify this avatar")
	}

	if err := user.UpdateUserEmail(db, req, userId); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if len(file) > 0 && ext != "" {
		avatarURL, err := UploadProfileImage(logger, db, userId, file, ext)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}

		req.AvatarURL = avatarURL
		req.AvatarUpdate = true
	}

	updatedProfile, err := userProfile.UpdateProfileFields(db, req, userId, logger)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	return http.StatusOK, updatedProfile, nil
}

func DeleteUserProfileImage(db *gorm.DB, logger *utility.Logger, userId string) (int, error) {
	var Profile models.Profile

	// Verify ownership before allowing avatar deletion
	// userId is the authenticated user ID from JWT, so we verify it matches and user exists
	isOwner, err := VerifyAvatarOwnership(db, userId, userId)
	if err != nil {
		logger.Error("Failed to verify avatar ownership", "error", err, "user_id", userId)
		return http.StatusInternalServerError, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if !isOwner {
		logUnauthorizedAvatarAccess(logger, userId, userId, "delete")
		return http.StatusForbidden, errors.New("you do not have permission to modify this avatar")
	}

	avatarURL, err := GetUserProfileImageURL(db, userId)
	if err != nil {
		logger.Error("Failed to retrieve user profile image", "error", err)
		return http.StatusInternalServerError, err
	}

	if avatarURL == "" {
		return http.StatusBadRequest, nil
	}

	err = DeleteUserProfileImageFromMinIO(logger, avatarURL)
	if err != nil {
		logger.Error("Failed to delete profile picture from MinIO", "error", err)
		return http.StatusInternalServerError, err
	}

	err = Profile.SetProfileImageToEmpty(db, userId)
	if err != nil {
		logger.Error("Failed to update user profile avatar URL in database", "error", err)
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func UploadProfileImage(logger *utility.Logger, db *gorm.DB, userID string, file []byte, ext string) (string, error) {
	if file != nil {
		picId := strings.Split(userID, "-")[4]
		filename := fmt.Sprintf("profile_pic_%s.%s", picId, ext)

		avatarURL, err := GetUserProfileImageURL(db, userID)
		if err != nil {
			return "", err
		}

		if avatarURL != "" {
			err = DeleteUserProfileImageFromMinIO(logger, avatarURL)
			if err != nil {
				return "", err
			}
		}

		picURL, err := minio.UploadProfilePic(logger, filename, bytes.NewReader(file), int64(len(file)))
		if err != nil {
			return "", err
		}

		return picURL, nil
	}

	return "", nil
}

func GetUserProfileImageURL(db *gorm.DB, userID string) (string, error) {
	var user models.User

	userProfile, err := user.GetUserWithProfile(db, userID)
	if err != nil {
		return "", err
	}

	if userProfile.Profile.AvatarURL == "" {
		return "", nil
	}

	return userProfile.Profile.AvatarURL, nil
}

func DeleteUserProfileImageFromMinIO(logger *utility.Logger, avatarURL string) error {
	urlParts := strings.Split(avatarURL, "/")
	objectName := urlParts[len(urlParts)-1]

	exists, err := minio.ProfileImageExists(logger, objectName)
	if err != nil {
		logger.Error("Failed to check if profile picture exists in MinIO", "error", err)
		return err
	}

	if !exists {
		logger.Info("Profile picture does not exist in MinIO, no deletion necessary", "objectName", objectName)
		return nil
	}

	err = minio.DeleteProfilePic(logger, objectName)
	if err != nil {
		logger.Error("Failed to delete profile picture from MinIO", "error", err)
		return err
	}

	logger.Info("Profile picture successfully deleted from MinIO", "objectName", objectName)
	return nil
}

func constructProfileSummary(userProfile models.User) *models.ProfileSummary {
	return &models.ProfileSummary{
		ID:                userProfile.Profile.ID,
		Email:             userProfile.Email,
		Phone:             userProfile.Profile.Phone,
		FirstName:         userProfile.Profile.FirstName,
		LastName:          userProfile.Profile.LastName,
		FullName:          userProfile.Profile.FullName,
		UserName:          userProfile.Profile.UserName,
		AvatarURL:         userProfile.Profile.AvatarURL,
		UserId:            userProfile.Profile.Userid,
		Deactivated:       userProfile.Deactivated,
		ProfileUpdated:    userProfile.ProfileUpdated,
		IsOnboarded:       userProfile.IsOnboarded,
		DisplayName:       userProfile.Profile.DisplayName,
		Title:             userProfile.Profile.Title,
		NamePronunciation: userProfile.Profile.NamePronunciation,
		Timezone:          userProfile.Profile.Timezone,
		CreatedAt:         userProfile.Profile.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         userProfile.Profile.UpdatedAt.Format(time.RFC3339),
		DeletedAt:         userProfile.Profile.DeletedAt.Time.Format(time.RFC3339),
		Icon:              userProfile.Profile.Icon,
		Text:              userProfile.Profile.Text,
		StatusTimeout:     userProfile.Profile.StatusTimeout,
		PauseNotification: userProfile.Profile.PauseNotification,
		WorkspaceID:       userProfile.Profile.WorkspaceID,
		Track:             userProfile.Profile.Track,
		Links:             []string(userProfile.Profile.Links),
	}
}

func UpdateProfileStatus(req models.UpdateProfileStatus, db *gorm.DB, logger *utility.Logger) (int, error) {
	var userProfile models.Profile

	if err := userProfile.UpdateProfileStatus(db, req); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusCreated, nil
}

// PartialUpdateProfileStatus applies a partial status patch atomically and returns the updated status.
func PartialUpdateProfileStatus(req models.PartialStatusUpdate, db *gorm.DB, logger *utility.Logger) (models.UserStatus, int, error) {
	var profile models.Profile

	if err := db.Where("userid = ?", req.UserID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.UserStatus{}, http.StatusNotFound, fmt.Errorf("profile not found for user")
		}
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to load profile: %w", err)
	}

	updates := map[string]any{}
	if req.Text != nil {
		updates["text"] = *req.Text
	}
	if req.Emoji != nil {
		updates["icon"] = *req.Emoji
	}
	if req.Expiry != nil {
		updates["status_timeout"] = strconv.FormatInt(*req.Expiry, 10)
	}
	if req.Visibility != nil {
		updates["status_visibility"] = *req.Visibility
	}

	if len(updates) == 0 {
		return models.UserStatus{}, http.StatusBadRequest, errors.New("no fields to update")
	}

	if err := db.Model(&models.Profile{}).
		Where("userid = ?", req.UserID).
		Updates(updates).Error; err != nil {
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to update status: %w", err)
	}

	if err := db.Where("userid = ?", req.UserID).First(&profile).Error; err != nil {
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to reload profile: %w", err)
	}

	expiry := int64(0)
	if profile.StatusTimeout != "" {
		if parsed, err := strconv.ParseInt(profile.StatusTimeout, 10, 64); err == nil {
			expiry = parsed
		}
	}

	visibility := "public"
	if profile.StatusVisibility != "" {
		visibility = profile.StatusVisibility
	}

	status := models.UserStatus{
		Text:       profile.Text,
		Emoji:      profile.Icon,
		Expiry:     expiry,
		Visibility: visibility,
	}

	if logger != nil {
		logger.Info("status updated", "user_id", req.UserID)
		notification := models.Notification[models.StatusUpdate]
		notification.SectionType = models.ChannelsSection
		notification.NotificationId = utility.GenerateUUID()
		notification.ModificationDetails = &models.ModificationDetails{
			UserId: req.UserID,
		}
		notification.Content = struct {
			UserID string            `json:"user_id"`
			Status models.UserStatus `json:"status"`
		}{
			UserID: req.UserID,
			Status: status,
		}

		channelID := fmt.Sprintf("user:%s", req.UserID)
		if err := centrifuge.PublishChannel(logger, channelID, notification); err != nil {
			logger.Error("failed to publish status update event", "error", err, "channel_id", channelID)
		}
	}

	return status, http.StatusOK, nil
}

// SetUserStatus sets a new status for a user and returns the saved status.
// Text is required; emoji, expiry, and visibility are optional.
func SetUserStatus(req models.SetStatusRequest, db *gorm.DB, logger *utility.Logger) (models.UserStatus, int, error) {
	var profile models.Profile

	if err := db.Where("userid = ?", req.UserID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.UserStatus{}, http.StatusNotFound, fmt.Errorf("profile not found for user")
		}
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to load profile: %w", err)
	}

	updates := map[string]any{
		"text": req.Text,
	}

	if req.Emoji != nil {
		updates["icon"] = *req.Emoji
	} else {
		updates["icon"] = ""
	}

	if req.Expiry != nil {
		updates["status_timeout"] = strconv.FormatInt(*req.Expiry, 10)
	} else {
		updates["status_timeout"] = ""
	}

	if req.Visibility != nil {
		updates["status_visibility"] = *req.Visibility
	} else {
		updates["status_visibility"] = "public"
	}

	if err := db.Model(&models.Profile{}).
		Where("userid = ?", req.UserID).
		Updates(updates).Error; err != nil {
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to update status: %w", err)
	}

	if err := db.Where("userid = ?", req.UserID).First(&profile).Error; err != nil {
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to reload profile: %w", err)
	}

	expiry := int64(0)
	if profile.StatusTimeout != "" {
		if parsed, err := strconv.ParseInt(profile.StatusTimeout, 10, 64); err == nil {
			expiry = parsed
		}
	}

	visibility := "public"
	if profile.StatusVisibility != "" {
		visibility = profile.StatusVisibility
	}

	status := models.UserStatus{
		Text:       profile.Text,
		Emoji:      profile.Icon,
		Expiry:     expiry,
		Visibility: visibility,
	}

	if logger != nil {
		logger.Info("status set", "user_id", req.UserID)
		notification := models.Notification[models.StatusUpdate]
		notification.SectionType = models.ChannelsSection
		notification.NotificationId = utility.GenerateUUID()
		notification.ModificationDetails = &models.ModificationDetails{
			UserId: req.UserID,
		}
		notification.Content = struct {
			UserID string            `json:"user_id"`
			Status models.UserStatus `json:"status"`
		}{
			UserID: req.UserID,
			Status: status,
		}

		channelID := fmt.Sprintf("user:%s", req.UserID)
		if err := centrifuge.PublishChannel(logger, channelID, notification); err != nil {
			logger.Error("failed to publish status update event", "error", err, "channel_id", channelID)
		}
	}

	return status, http.StatusCreated, nil
}

// GetUserStatus retrieves the current status for a user.
// Returns a UserStatus object with default values if no status is set or profile not found.
func GetUserStatus(userID string, db *gorm.DB) (models.UserStatus, int, error) {
	var profile models.Profile

	// Query profile by userid
	if err := db.Where("userid = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty status with defaults if profile not found
			return models.UserStatus{
				Text:       "",
				Emoji:      "",
				Expiry:     0,
				Visibility: "public",
			}, http.StatusOK, nil
		}
		return models.UserStatus{}, http.StatusInternalServerError, fmt.Errorf("failed to load profile: %w", err)
	}

	// Parse StatusTimeout string to int64 expiry timestamp
	expiry := int64(0)
	if profile.StatusTimeout != "" {
		if parsed, err := strconv.ParseInt(profile.StatusTimeout, 10, 64); err == nil {
			expiry = parsed
		}
	}

	// Get visibility from profile, default to "public" if empty
	visibility := "public"
	if profile.StatusVisibility != "" {
		visibility = profile.StatusVisibility
	}

	// Return status with visibility from database
	status := models.UserStatus{
		Text:       profile.Text,
		Emoji:      profile.Icon,
		Expiry:     expiry,
		Visibility: visibility,
	}

	return status, http.StatusOK, nil
}
