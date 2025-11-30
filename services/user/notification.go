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

func GetUserNotificationPreferences(db *gorm.DB, c *gin.Context) (*models.NotificationPreferences, int, error) {
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

func UpdateUserNotificationPreferences(userData models.NotificationPreferences,
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

func GetEffectiveNotificationSettings(db *gorm.DB, c *gin.Context) (*models.EffectiveNotificationSetting, int, error) {
	var (
		userNotiData models.UserNotificationSetting
		theUserData  models.UserNotificationSetting

		deviceNotiData models.DeviceNotificationSetting
		theDeviceData  models.DeviceNotificationSetting

		effecNotiData models.EffectiveNotificationSetting
	)

	// Get UserID from JWT claims
	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	userID, err := uuid.FromString(currentUserID)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	// Verify User exists
	_, code, err := GetUser(currentUserID, db)
	if err != nil {
		return nil, code, err
	}

	if theUserData, err = userNotiData.GetUserNotificationSettingByID(db, currentUserID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.UserNotificationSetting{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.CreateUserNotificationSetting(db)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}

			if userNotiData, err = userNotiData.GetUserNotificationSettingByID(db, currentUserID); err != nil {
				return nil, http.StatusBadRequest, err
			}
		}
		return nil, http.StatusBadRequest, err
	}

	// Get Device ID from the query parameter
	deviceID := c.Query("device_id")

	// Verify Device ID is not empty
	if deviceID == "" {
		return nil, http.StatusBadRequest, errors.New("device_id is required")
	}

	// Get the Device Notification
	if theDeviceData, err = deviceNotiData.GetDeviceNotificationSettingByID(db, currentUserID, deviceID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.DeviceNotificationSetting{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.CreateDeviceNotificationSetting(db)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}

			if theDeviceData, err = deviceNotiData.GetDeviceNotificationSettingByID(db, currentUserID, deviceID); err != nil {
				return nil, http.StatusBadRequest, err
			}
		}
		return nil, http.StatusBadRequest, err
	}

	effecNotiData = models.MergePreferences(&theDeviceData, &theUserData)
	effecNotiData.DeviceID = deviceID

	return &effecNotiData, http.StatusOK, nil
}

func UpdateUserNotificationSetting(userInputNotification models.NotificationSettingsUpdateRequest,
	db *gorm.DB, c *gin.Context) (*models.EffectiveNotificationSetting, int, error) {
	var (
		userNotiData models.UserNotificationSetting
		theUserData  models.UserNotificationSetting

		deviceNotiData models.DeviceNotificationSetting
		theDeviceData  models.DeviceNotificationSetting

		effectiveNotiData models.EffectiveNotificationSetting
	)

	// Get UserID from JWT claims
	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	userID, err := uuid.FromString(currentUserID)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("invalid thread ID")
	}

	// Verify User exists
	_, code, err := GetUser(currentUserID, db)
	if err != nil {
		return nil, code, err
	}

	// Get User's Notification Settings
	if theUserData, err = userNotiData.GetUserNotificationSettingByID(db, currentUserID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.UserNotificationSetting{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.CreateUserNotificationSetting(db)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}

			if userNotiData, err = userNotiData.GetUserNotificationSettingByID(db, currentUserID); err != nil {
				return nil, http.StatusBadRequest, err
			}
		}
		return nil, http.StatusBadRequest, err
	}

	// Get Device ID from the query parameter
	deviceID := c.Query("device_id")

	// Verify Device ID is not empty
	if deviceID == "" {
		return nil, http.StatusBadRequest, errors.New("device_id is required")
	}

	// Get the Device Notification Setting
	if theDeviceData, err = deviceNotiData.GetDeviceNotificationSettingByID(db, currentUserID, deviceID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.DeviceNotificationSetting{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.CreateDeviceNotificationSetting(db)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}

			if theDeviceData, err = deviceNotiData.GetDeviceNotificationSettingByID(db, currentUserID, deviceID); err != nil {
				return nil, http.StatusBadRequest, err
			}
		}
		return nil, http.StatusBadRequest, err
	}

	err = MergeNotificationRequest(
		userInputNotification,
		&deviceNotiData,
		&userNotiData,
		db,
	)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	effectiveNotiData = models.MergePreferences(&theDeviceData, &theUserData)
	return &effectiveNotiData, http.StatusOK, nil

}

func ResetNotificationSettings(db *gorm.DB, c *gin.Context) (int, error) {
	defaultNotifcationSetting := models.DefaultNotificationPreferences()

	var (
		userNotifData models.UserNotificationSetting
		theUserData   models.UserNotificationSetting

		deviceNotifData models.DeviceNotificationSetting
		theDeviceData   models.DeviceNotificationSetting
	)

	// Get UserID from JWT claims
	userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return http.StatusNotFound, err
	}

	currentUserID, ok := userId.(string)
	if !ok {
		return http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	userID, err := uuid.FromString(currentUserID)
	if err != nil {
		return http.StatusBadRequest, errors.New("invalid thread ID")
	}

	// Verify User exists
	_, code, err := GetUser(currentUserID, db)
	if err != nil {
		return code, err
	}

	// Get User's Notification Settings
	if theUserData, err = userNotifData.GetUserNotificationSettingByID(db, currentUserID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.UserNotificationSetting{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.CreateUserNotificationSetting(db)
			if err != nil {
				return http.StatusBadRequest, err
			}

			if userNotifData, err = userNotifData.GetUserNotificationSettingByID(db, currentUserID); err != nil {
				return http.StatusBadRequest, err
			}
		}
		return http.StatusBadRequest, err
	}

	// Get Device ID from the query parameter
	deviceID := c.Query("device_id")

	// Verify Device ID is not empty
	if deviceID == "" {
		return http.StatusBadRequest, errors.New("device_id is required")
	}

	// Get the Device Notification Setting
	if theDeviceData, err = deviceNotifData.GetDeviceNotificationSettingByID(db, currentUserID, deviceID); err != nil {
		if err.Error() == "record not found" {
			theModel := models.DeviceNotificationSetting{
				ID:     utility.GenerateUUID(),
				UserID: userID,
			}

			theModel.UserID = userID
			err = theModel.CreateDeviceNotificationSetting(db)
			if err != nil {
				return http.StatusBadRequest, err
			}

			if theDeviceData, err = deviceNotifData.GetDeviceNotificationSettingByID(db, currentUserID, deviceID); err != nil {
				return http.StatusBadRequest, err
			}
		}
		return http.StatusBadRequest, err
	}

	theUserData.NotificationSettings = defaultNotifcationSetting
	err = theUserData.UpdateUserNotificationSetting(db)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	theDeviceData.NotificationSettings = defaultNotifcationSetting
	err = theDeviceData.UpdateDeviceNotificationSetting(db)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func MergeNotificationRequest(
	userInputNotification models.NotificationSettingsUpdateRequest,
	deviceNotificationSetting *models.DeviceNotificationSetting,
	userNotificationSetting *models.UserNotificationSetting,
	db *gorm.DB,
) error {
	// Merge userInput message notification Request with deviceNotificationSetting
	if userInputNotification.MessageNotification != nil {
		deviceNotificationSetting.MessageNotification = userNotificationSetting.MessageNotification
	}

	// Merge userInput group notification Request with deviceNotificationSetting
	if userInputNotification.GroupNotification != nil {
		deviceNotificationSetting.GroupNotification = userNotificationSetting.GroupNotification
	}

	// Merge userInput reminder notification Request with userNotificationSetting
	if userInputNotification.Reminders != nil {
		deviceNotificationSetting.Reminders = userNotificationSetting.Reminders
	}

	// Merge userInput InAppNotification Request with deviceNotificationSetting
	if userInputNotification.InAppNotifications != nil {
		deviceNotificationSetting.InAppNotifications = userNotificationSetting.InAppNotifications
	}

	// Merge userInput showPreview notification Request with deviceNotificationSetting
	if userInputNotification.ShowPreview != nil {
		deviceNotificationSetting.ShowPreview = userNotificationSetting.ShowPreview
	}

	// Save UserNotificationSettings In Db
	err := userNotificationSetting.UpdateUserNotificationSetting(db)
	if err != nil {
		return err
	}

	// Save UserNotificationSettings In Db
	err = deviceNotificationSetting.UpdateDeviceNotificationSetting(db)
	if err != nil {
		return err
	}

	return nil
}
