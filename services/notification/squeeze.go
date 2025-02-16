package notifications

import (
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendSqueeze() error {
	var (
		notificationData     = models.SendSqueeze{}
		templateFileName     = "squeeze.html"
		baseTemplateFileName = ""
		subject              = "Subject: Welcome to Our Service"
		configData           = config.GetConfig()
	)
	signUpUrl := fmt.Sprintf("%v/auth/sign-up", configData.App.FRONTEND_URL)
	faqUrl := fmt.Sprintf("%v/faq", configData.App.FRONTEND_URL)
	contactUrl := fmt.Sprintf("%v/contact", configData.App.FRONTEND_URL)
	policyUrl := fmt.Sprintf("%v/policy", configData.App.FRONTEND_URL)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]interface{}{
		"firstname": thisOrThatStr(notificationData.FirstName, notificationData.Email),
		"sign_up_url": signUpUrl,
		"faq_url": faqUrl,
		"contact_us_url": contactUrl,
		"privacy_policy_url": policyUrl,
	})
	if err != nil {
		return fmt.Errorf("error converting data to map, %v", err)
	}

	return send.SendEmail(n.ExtReq, notificationData.Email, subject, templateFileName, baseTemplateFileName, data)
}
