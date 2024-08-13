package profile

import (
	"bufio"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
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
		CreatedAt: userProfile.Profile.CreatedAt.Format(time.RFC3339),
		UpdatedAt: userProfile.Profile.UpdatedAt.Format(time.RFC3339),
		DeletedAt: userProfile.Profile.DeletedAt.Time.Format(time.RFC3339),
	}

	return &profileSummary, http.StatusOK, nil
}

func UpdateUserProfile(req models.UpdateUserProfileRequest, db *gorm.DB, logger *utility.Logger, userId string, header *multipart.FileHeader, file multipart.File) (int, error) {
	var user models.User
	var userProfile models.Profile

	if err := user.UpdateUserEmail(db, req, userId); err != nil {
		return http.StatusInternalServerError, err
	}

	if file != nil {
		fileType := filepath.Ext(header.Filename)
		picId := strings.Split(userId, "-")[4]
		filename := fmt.Sprintf("profile_pic_%s%s", picId, fileType)

		picUrl, err := minio.UploadProfilePic(logger, filename, bufio.NewReader(file), header.Size)

		if err != nil {
			return http.StatusBadRequest, err
		}
		req.AvatarURL = picUrl

		defer file.Close()
	}

	if err := userProfile.UpdateProfileFields(db, req, userId); err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}
