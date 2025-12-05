package profile

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
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

	// Verify that the user exists in the database
	var user models.User
	exists := db.Where("id = ?", authenticatedUserID).First(&user).Error
	if exists != nil {
		if errors.Is(exists, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, exists
	}

	return true, nil
}

// logUnauthorizedAvatarAccess logs unauthorized avatar access attempts for security monitoring
func logUnauthorizedAvatarAccess(logger *utility.Logger, authenticatedUserID, targetUserID, operation string) {
	logger.Error(
		"Unauthorized avatar access attempt",
		"authenticated_user_id", authenticatedUserID,
		"target_user_id", targetUserID,
		"operation", operation,
		"timestamp", time.Now().Format(time.RFC3339),
	)
}

func UpdateUserProfile(req models.UpdateUserProfileRequest, db *gorm.DB, logger *utility.Logger, userId string, ext string, file []byte) (int, error) {
	var user models.User
	var userProfile models.Profile

	// Verify ownership before allowing any avatar operations
	isOwner, err := VerifyAvatarOwnership(db, userId, userId)
	if err != nil {
		logger.Error("Failed to verify avatar ownership", "error", err, "user_id", userId)
		return http.StatusInternalServerError, errors.New("failed to verify avatar ownership")
	}
	if !isOwner {
		logUnauthorizedAvatarAccess(logger, userId, userId, "update")
		return http.StatusForbidden, errors.New("you do not have permission to modify this avatar")
	}

	if err := user.UpdateUserEmail(db, req, userId); err != nil {
		return http.StatusInternalServerError, err
	}

	if len(file) > 0 && ext != "" {
		avatarURL, err := UploadProfileImage(logger, db, userId, file, ext)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		req.AvatarURL = avatarURL
		req.AvatarUpdate = true
	}

	if err := userProfile.UpdateProfileFields(db, req, userId, logger); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func DeleteUserProfileImage(db *gorm.DB, logger *utility.Logger, userId string) (int, error) {
	var Profile models.Profile

	// Verify ownership before allowing avatar deletion
	isOwner, err := VerifyAvatarOwnership(db, userId, userId)
	if err != nil {
		logger.Error("Failed to verify avatar ownership", "error", err, "user_id", userId)
		return http.StatusInternalServerError, errors.New("failed to verify avatar ownership")
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
		filename := fmt.Sprintf("profile_pic_%s", picId)

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

	exists, err := minio.ImageExists(logger, objectName)
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
