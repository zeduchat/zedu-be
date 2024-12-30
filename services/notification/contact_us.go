package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendContactUsMail() error {
	var (
		notificationData     = models.SendContactUsMail{}
		subject              = "Subject: Thank You for Contacting Us😇"
		templateFileName     = "contact_us.html"
		baseTemplateFileName = ""
		configData           = config.GetConfig()
	)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	helpCenterUrl := fmt.Sprintf("%v/help", configData.App.FRONTEND_URL)
	loginUrl := fmt.Sprintf("%v/auth/login", configData.App.FRONTEND_URL)

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]interface{}{
		"firstname":       thisOrThatStr(notificationData.Name, notificationData.Email),
		"phone_number":    notificationData.PhoneNumber,
		"email":           notificationData.Email,
		"message":         notificationData.Message,
		"help_center_url": helpCenterUrl,
		"login_url":       loginUrl,
	})

	if err != nil {
		return fmt.Errorf("error converting data to map, %v, %v", err, strings.Join([]string{err.Error()}, ", "))
	}

	return send.SendEmail(n.ExtReq, notificationData.Email, subject, templateFileName, baseTemplateFileName, data)
}
