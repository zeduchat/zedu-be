package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendMagicLink() error {
	var (
		notificationData     = models.SendMagicLink{}
		templateFileName     = "send_magic_link.html"
		baseTemplateFileName = ""
		errs                 []string
		user                 models.User
		configData           = config.GetConfig()
	)
	subject := "Subject: Secure Login: Your MagicLink..."
	loginUrl := fmt.Sprintf("%v/login", configData.App.FRONTEND_URL)
	contactUrl := fmt.Sprintf("%v/contact", configData.App.FRONTEND_URL)
	policyUrl := fmt.Sprintf("%v/policy", configData.App.FRONTEND_URL)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	user, err = user.GetUserByEmail(n.Db, notificationData.Email)
	if err != nil {
		return fmt.Errorf("error getting user with account id %v, %v", notificationData.Email, err)
	}

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]any{
		"firstname":          thisOrThatStr(user.Profile.FirstName, user.Email),
		"business_name":      thisOrThatStr("", ""),
		"login_url":          loginUrl,
		"contact_us_url":     contactUrl,
		"privacy_policy_url": policyUrl,
	})
	if err != nil {
		return fmt.Errorf("error converting data to map, %v", err)
	}

	err = send.SendEmail(n.ExtReq, user.Email, subject, templateFileName, baseTemplateFileName, data)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}
	return nil
}
