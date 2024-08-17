package profile

import (
	"bytes"
	"encoding/base64"
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

	profileSummary := models.ProfileSummary{
		ID:        userProfile.Profile.ID,
		Email:     userProfile.Email,
		Phone:     userProfile.Profile.Phone,
		FirstName: userProfile.Profile.FirstName,
		LastName:  userProfile.Profile.LastName,
		FullName:  userProfile.Profile.FullName,
		UserName:  userProfile.Profile.UserName,
		AvatarURL: userProfile.Profile.AvatarURL,
		UserId:    userProfile.Profile.Userid,
		Deactivated: userProfile.Deactivated,
		CreatedAt: userProfile.Profile.CreatedAt.Format(time.RFC3339),
		UpdatedAt: userProfile.Profile.UpdatedAt.Format(time.RFC3339),
		DeletedAt: userProfile.Profile.DeletedAt.Time.Format(time.RFC3339),
	}

	return &profileSummary, http.StatusOK, nil
}

func UpdateUserProfile(req models.UpdateUserProfileRequest, db *gorm.DB, logger *utility.Logger, userId string, ext string, file []byte) (int, error) {
	var user models.User
	var userProfile models.Profile

	if err := user.UpdateUserEmail(db, req, userId); err != nil {
		return http.StatusInternalServerError, err
	}

	if file != nil {
		picId := strings.Split(userId, "-")[4]
		filename := fmt.Sprintf("profile_pic_%s%s", picId, ext)

		picUrl, err := minio.UploadProfilePic(logger, filename, bytes.NewReader(file), int64(len(file)))

		if err != nil {
			return http.StatusBadRequest, err
		}
		req.AvatarURL = picUrl
	}

	if err := userProfile.UpdateProfileFields(db, req, userId); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func ValidatePicture(base64Image string) ([]byte, string, error) {
	const maxImageSize = 5 * 1024 * 1024 // 5MB

	var (
		imageData []byte
		ext       string
	)

	if base64Image == "" {
		return nil, "", nil
	}

	if isURLPrefix(base64Image) {
    	return nil, "", nil
	}

	switch {
	case strings.HasPrefix(base64Image, "data:image/jpeg;base64,"):
		ext = ".jpeg"
	case strings.HasPrefix(base64Image, "data:image/jpg;base64,"):
		ext = ".jpg"
	case strings.HasPrefix(base64Image, "data:image/png;base64,"):
		ext = ".png"
	default:
		return nil, "", fmt.Errorf("invalid content type: only PNG, JPEG, or JPG images are allowed")
	}

	if len(imageData) > maxImageSize {
		return imageData, ext, fmt.Errorf("image size exceeds 5MB limit")
	}

	parts := strings.SplitN(base64Image, ",", 2)

	if len(parts) < 2 {
		return imageData, ext, fmt.Errorf("invalid data URL")
	}
	base64ImageData := parts[1]

	imageData, err := base64.StdEncoding.DecodeString(base64ImageData)
	if err != nil {
		return imageData, ext, fmt.Errorf("failed to decode base64 string: %w", err)
	}

	return imageData, ext, nil
}

func DeleteUserProfileImage(db *gorm.DB, logger *utility.Logger, userId string) (int, error) {
	var userProfile models.Profile

	if err := userProfile.ReplaceAvatarWithDefault(db, userId); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func isURLPrefix(url string) bool {
    supportedPrefixes := []string{"http://", "https://", "blob:", "ipfs://", "ftp:"}
    for _, prefix := range supportedPrefixes {
        if strings.HasPrefix(url, prefix) {
            return true
        }
    }
    return false
}