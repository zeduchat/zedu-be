package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func GetUserNotificationSettings(db *gorm.DB, c *gin.Context) (*models.NotificationPreferences, int, error) {
	var (
		notiData models.NotificationPreferences
		theData  models.NotificationPreferences
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return &theData, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return &theData, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	userID, err := uuid.FromString(currentUserID)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	_, code, err := GetUser(currentUserID, db)
	if err != nil {
		return &theData, code, err
	}

	if theData, err = notiData.GetUserDataByID(db, currentUserID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.NotificationPreferences{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.Create(db)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}

			if theData, err = notiData.GetUserDataByID(db, currentUserID); err != nil {
				return nil, http.StatusBadRequest, err
			}
			return &theData, http.StatusOK, nil
		}
		return &theData, http.StatusBadRequest, err
	}

	return &theData, http.StatusOK, nil
}

func UpdateUserNotificationSettings(userData models.NotificationPreferences,
	db *gorm.DB, c *gin.Context) (*models.NotificationPreferences, int, error) {
	var (
		notiData models.NotificationPreferences
		theData  models.NotificationPreferences
	)

	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return &theData, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return &theData, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	_, code, err := GetUser(currentUserID, db)
	if err != nil {
		return &theData, code, err
	}

	if theData, err = notiData.GetUserDataByID(db, currentUserID); err != nil {
		return &theData, http.StatusBadRequest, err
	} else {

		theData.NotifyAbout = userData.NotifyAbout
		theData.NotificationSchedule = userData.NotificationSchedule
		theData.FromHour = userData.FromHour
		theData.ToHour = userData.ToHour
		theData.NotificationMethodEmail = userData.NotificationMethodEmail

		if err := theData.Update(db); err != nil {
			return &theData, http.StatusBadRequest, err
		}
		return &theData, http.StatusOK, nil
	}

}
