package profile

import (
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)


func GetUserProfile( db *gorm.DB, userID string) (*models.ProfileSummary, int, error) {
    var user models.User
	userProfile, err := user.GetUserWithProfile(db, userID)
	
	if err != nil {
         return nil, http.StatusNotFound, err
	}

	profileSummary := models.ProfileSummary{
		ID:        userProfile.Profile.ID,
		Email:     userProfile.Email,
		Phone:     userProfile.Profile.Phone,
		FirstName:  userProfile.Profile.FirstName,
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


func UpdateUserProfile(db *gorm.DB, req models.UpdateUserProfileRequest, userId string) (map[string]interface{}, int, error) {
	var user models.User
	var profile models.Profile

	profileId, err := user.GetProfileID(db, userId)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	if err := user.UpdateUserProfileEmail(db, req, userId); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if err := profile.UpdateProfileFields(db, req, profileId); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	responseData := models.PrepareResponseData(req)
	return responseData, http.StatusOK, nil
}