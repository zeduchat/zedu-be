package notifications

import (
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendLoginAlertMail() error {
	var (
		notificationData     models.SendLoginAlertMail
		subject              = "Recent Login"
		templateFileName     = "login_alert.html"
		baseTemplateFileName = ""
		configData           = config.GetConfig()
		user                 models.User
	)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data: %v", err)
	}

	passwordResetUrl := fmt.Sprintf("%v/reset-password/", configData.App.Url)

	user, err = user.GetUserByEmail(n.Db, notificationData.Email)
	if err != nil {
		return fmt.Errorf("error getting user with email %v: %v", notificationData.Email, err)
	}

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]interface{}{
		"firstname":         thisOrThatStr(user.Profile.FirstName, user.Email),
		"business_name":     thisOrThatStr("", ""),
		"password_reset_url": passwordResetUrl,
	})
	if err != nil {
		return fmt.Errorf("error converting data to map: %v", err)
	}

	return send.SendEmail(n.ExtReq, notificationData.Email, subject, templateFileName, baseTemplateFileName, data)
}
