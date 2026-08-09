package profile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func GetUserProfile(db *gorm.DB, userID string, orgID ...string) (*models.ProfileSummary, int, error) {
	var targetOrg string
	if len(orgID) > 0 {
		targetOrg = orgID[0]
	}

	var profModel models.Profile
	profile, err := profModel.GetOrCreateProfileForOrg(db, userID, targetOrg)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	var user models.User
	userObj, _ := user.GetUserByID(db, userID, targetOrg)

	profileSummary := constructProfileSummaryWithProfile(userObj, profile)
	return profileSummary, http.StatusOK, nil
}

func IsSameOrganization(db *gorm.DB, orgID string, reqUserID string, targetUserID string) (int, error) {
	if orgID == "" {
		return http.StatusBadRequest, errors.New("organisation ID is required")
	}

	var org models.Organisation
	if reqUserID != "" {
		isReqUserMember, err := org.CheckUserIsMemberOfOrg(reqUserID, orgID, db)
		if err != nil || !isReqUserMember {
			return http.StatusBadRequest, errors.New("user not authorised to retrieve this organisation")
		}
	}

	isMember, err := org.CheckUserIsMemberOfOrg(targetUserID, orgID, db)
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

func UpdateUserProfile(req models.UpdateUserProfileRequest, db *gorm.DB, logger *utility.Logger, userId string, ext string, file []byte, orgId ...string) (int, *models.Profile, error) {
	var user models.User
	var userProfile models.Profile

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

	var targetOrg string
	if len(orgId) > 0 {
		targetOrg = orgId[0]
	}

	if len(file) > 0 && ext != "" {
		avatarURL, err := UploadProfileImage(logger, db, userId, file, ext, targetOrg)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}

		req.AvatarURL = avatarURL
		req.AvatarUpdate = true
	}

	updatedProfile, err := userProfile.UpdateProfileFields(db, req, userId, logger, targetOrg)
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	logger.Info("User profile updated", "user_id", userId)
	notification := models.Notification[models.ProfileUpdated]
	notification.SectionType = models.OrganisationUsersSection
	notification.NotificationId = utility.GenerateUUID()
	notification.ModificationDetails = &models.ModificationDetails{
		UserId: userId,
	}
	notification.Content = struct {
		UserID  string         `json:"user_id"`
		Profile models.Profile `json:"profile"`
	}{
		UserID:  userId,
		Profile: *updatedProfile,
	}

	if err := centrifuge.PublishChannel(logger, targetOrg, notification); err != nil {
		logger.Error("failed to publish status update event", "error", err, "organisation_id", targetOrg)
	}

	logger.Info("Published profile update for,", "user_id", userId, "organisation_id", targetOrg)

	return http.StatusOK, updatedProfile, nil
}

func DeleteUserProfileImage(db *gorm.DB, logger *utility.Logger, userId string, orgId ...string) (int, error) {
	var Profile models.Profile

	isOwner, err := VerifyAvatarOwnership(db, userId, userId)
	if err != nil {
		logger.Error("Failed to verify avatar ownership", "error", err, "user_id", userId)
		return http.StatusInternalServerError, fmt.Errorf("failed to verify ownership: %w", err)
	}
	if !isOwner {
		logUnauthorizedAvatarAccess(logger, userId, userId, "delete")
		return http.StatusForbidden, errors.New("you do not have permission to modify this avatar")
	}

	var targetOrg string
	if len(orgId) > 0 {
		targetOrg = orgId[0]
	}

	avatarURL, err := GetUserProfileImageURL(db, userId, targetOrg)
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

	err = Profile.SetProfileImageToEmpty(db, userId, logger, targetOrg)
	if err != nil {
		logger.Error("Failed to update user profile avatar URL in database", "error", err)
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func UploadProfileImage(logger *utility.Logger, db *gorm.DB, userID string, file []byte, ext string, orgId ...string) (string, error) {
	if file != nil {
		var targetOrg string
		if len(orgId) > 0 {
			targetOrg = orgId[0]
		}

		filename := fmt.Sprintf("profile_pic_%s_%d.%s", userID, time.Now().UnixNano(), ext)

		avatarURL, err := GetUserProfileImageURL(db, userID, targetOrg)
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

func GetUserProfileImageURL(db *gorm.DB, userID string, orgId ...string) (string, error) {
	var targetOrg string
	if len(orgId) > 0 {
		targetOrg = orgId[0]
	}

	var profModel models.Profile
	prof, err := profModel.GetOrCreateProfileForOrg(db, userID, targetOrg)
	if err != nil {
		return "", err
	}

	return prof.AvatarURL, nil
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

func constructProfileSummaryWithProfile(userObj models.User, prof models.Profile) *models.ProfileSummary {
	return &models.ProfileSummary{
		ID:                prof.ID,
		Email:             userObj.Email,
		Phone:             prof.Phone,
		FirstName:         prof.FirstName,
		LastName:          prof.LastName,
		FullName:          prof.FullName,
		UserName:          prof.UserName,
		AvatarURL:         prof.AvatarURL,
		DefaultAvatarURL:  avatar.GenerateDefaultAvatarURL(prof.Userid),
		UserId:            prof.Userid,
		OrganisationID:    prof.GetOrgID(),
		Deactivated:       userObj.Deactivated,
		ProfileUpdated:    userObj.ProfileUpdated,
		IsOnboarded:       userObj.IsOnboarded,
		DisplayName:       prof.DisplayName,
		Title:             prof.Title,
		NamePronunciation: prof.NamePronunciation,
		Timezone:          prof.Timezone,
		CreatedAt:         prof.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         prof.UpdatedAt.Format(time.RFC3339),
		DeletedAt:         prof.DeletedAt.Time.Format(time.RFC3339),
		Icon:              prof.Icon,
		Text:              prof.Text,
		StatusTimeout:     prof.StatusTimeout,
		PauseNotification: prof.PauseNotification,
		WorkspaceID:       prof.WorkspaceID,
		Track:             prof.Track,
		Links:             []string(prof.Links),
		Online:            prof.Online,
	}
}

func constructProfileSummary(userProfile models.User) *models.ProfileSummary {
	return constructProfileSummaryWithProfile(userProfile, userProfile.Profile)
}

// GetUserStatus retrieves the current status for a user.
// Returns a UserStatus object with default values if no status is set or profile not found.
func GetUserStatus(userID string, db *gorm.DB, orgID ...string) (models.UserStatus, int, error) {
	var profModel models.Profile
	var targetOrg string
	if len(orgID) > 0 {
		targetOrg = orgID[0]
	}

	profile, err := profModel.GetOrCreateProfileForOrg(db, userID, targetOrg)
	if err != nil {
		return models.UserStatus{
			Text:       "",
			Emoji:      "",
			Expiry:     0,
			Visibility: "public",
			Online:     false,
		}, http.StatusOK, nil
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
		Online:     profile.Online,
	}

	return status, http.StatusOK, nil
}

func UpdateUserPresence(req models.UpdateUserPresenceRequest, db *gorm.DB, logger *utility.Logger) (int, error) {
	var profModel models.Profile
	profile, err := profModel.GetOrCreateProfileForOrg(db, req.UserID, req.OrgID)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("profile not found")
	}

	if err := db.Model(&models.Profile{}).Where("id = ?", profile.ID).Update("online", req.IsActive).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update presence: %w", err)
	}

	if logger != nil {
		logger.Info("presence updated", "user_id", req.UserID, "is_active", req.IsActive)

		notification := models.Notification[models.UserPresenceChanged]
		notification.NotificationId = utility.GenerateUUID()
		notification.Content = models.UserPresenceChangedPayload{
			UserID: req.UserID,
			Online: req.IsActive,
		}
		notification.ModificationDetails = &models.ModificationDetails{
			UserId: req.UserID,
			OrgId:  req.OrgID,
		}

		// Broadcast to organization channel so other users can see the update
		channelID := req.OrgID
		if err := centrifuge.PublishChannel(logger, channelID, notification); err != nil {
			logger.Error("failed to publish presence update event", "error", err, "channel_id", channelID)
		}
	}

	return http.StatusOK, nil
}

func GetUserPresence(userID, orgID string, db *gorm.DB) (bool, int, error) {
	var profile models.Profile

	profile, err := profile.GetOrCreateProfileForOrg(db, userID, orgID)

	if err != nil {
		if err := db.Select("online").Where("userid = ?", userID).First(&profile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, http.StatusNotFound, fmt.Errorf("profile not found")
			}
			return false, http.StatusInternalServerError, fmt.Errorf("failed to fetch presence: %w", err)
		}
	}

	return profile.Online, http.StatusOK, nil
}

// UpdateProfileStatusWithJobScheduling updates a user's profile status and handles River job scheduling.
func UpdateProfileStatusWithJobScheduling(req models.UpdateProfileStatus, db *storage.Database, logger *utility.Logger) (models.UserStatus, int, error) {
	var profileModel models.Profile
	var gormDB *gorm.DB
	if db != nil {
		gormDB = db.Postgresql
	}

	status, code, err := profileModel.UpdateProfileStatus(gormDB, req, logger)
	if err != nil {
		return status, code, err
	}

	statusTimeout := req.StatusTimeout
	if statusTimeout == "" {
		statusTimeout = req.StatusExpiry
	}

	// Clear river_job_id if no expiry provided (cancels any existing auto-clear job)
	if statusTimeout == "" {
		riverClient := db.River
		if riverClient == nil && storage.DB != nil {
			riverClient = storage.DB.River
		}
		if profileModel.RiverJobID != nil && riverClient != nil {
			ctx := context.Background()
			_, cancelErr := riverClient.JobCancel(ctx, *profileModel.RiverJobID)
			if cancelErr != nil {
				logger.Error("failed to cancel existing clear status job %d: %v", *profileModel.RiverJobID, cancelErr)
			}
		}

		if err := gormDB.Model(&models.Profile{}).Where("id = ?", profileModel.ID).Update("river_job_id", nil).Error; err != nil {
			logger.Error("failed to clear river_job_id: %v", err)
		}
	}

	if statusTimeout != "" {
		var expiryTimestamp int64
		expiryTimestamp, err = profileModel.ParseStatusExpiry(statusTimeout)
		if err != nil {
			return models.UserStatus{}, http.StatusBadRequest, err
		}

		if expiryTimestamp <= 0 {
			logger.Info("Skipping clear status job scheduling for user %s: expiryTimestamp is %d (StatusTimeout: '%s' - likely 'don't remove')", req.UserId, expiryTimestamp, req.StatusTimeout)
		}

		if expiryTimestamp > 0 {
			riverClient := db.River
			if riverClient == nil && storage.DB != nil {
				riverClient = storage.DB.River
			}
			if profileModel.RiverJobID != nil && riverClient != nil {
				ctx := context.Background()
				_, cancelErr := riverClient.JobCancel(ctx, *profileModel.RiverJobID)
				if cancelErr != nil {
					logger.Error("failed to cancel existing clear status job %d: %v", *profileModel.RiverJobID, cancelErr)
				}
			}

			jobArgs := &models.ClearUserStatusJobArgs{
				UserID: req.UserId,
				OrgID:  req.OrgId,
			}
			scheduledAt := time.Unix(expiryTimestamp, 0)
			logger.Info("Scheduling clear status job for user %s at %s (OrgID: %s)", req.UserId, scheduledAt.Format(time.RFC3339), req.OrgId)
			insertRes, jobErr := InsertClearStatusJob(context.Background(), db, jobArgs, scheduledAt)
			if jobErr != nil {
				logger.Error("failed to insert clear status job for user %s: %v", req.UserId, jobErr)
			} else if insertRes != nil {
				logger.Info("Successfully scheduled clear status job %d for user %s", insertRes.Job.ID, req.UserId)
				if err := gormDB.Model(&models.Profile{}).Where("id = ?", profileModel.ID).Update("river_job_id", insertRes.Job.ID).Error; err != nil {
					logger.Error("failed to update river_job_id: %v", err)
				}
			} else {
				logger.Error("failed to insert clear status job for user %s: insert result is nil", req.UserId)
			}
		} else {
			if err := gormDB.Model(&models.Profile{}).Where("id = ?", profileModel.ID).Update("river_job_id", nil).Error; err != nil {
				logger.Error("failed to clear river_job_id: %v", err)
			}
		}
	} else {
		logger.Info("Skipping clear status job scheduling for user %s: StatusExpiry is empty", req.UserId)
	}

	return status, code, nil
}

// InsertClearStatusJob schedules a job to clear user status after it expires.
func InsertClearStatusJob(ctx context.Context, db *storage.Database, args *models.ClearUserStatusJobArgs, scheduledAt time.Time) (*rivertype.JobInsertResult, error) {
	if db == nil {
		db = storage.DB
	}
	if db == nil || db.River == nil {
		return nil, fmt.Errorf("river client is not initialized")
	}
	insertRes, err := db.River.Insert(ctx, args, &river.InsertOpts{
		MaxAttempts: 3,
		ScheduledAt: scheduledAt,
		Priority:    3,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert clear status job: %w", err)
	}
	return insertRes, nil
}
