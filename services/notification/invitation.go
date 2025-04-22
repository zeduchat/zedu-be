package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendInvitationLink() error {
	var (
		notificationData     = models.SendInvitationLink{}
		templateFileName     = "send_invitation.html"
		baseTemplateFileName = ""
		errs                 []string
		configData           = config.GetConfig()
	)
	subject := "Subject: Secure Login: Your Invitation..."
	contactUrl := fmt.Sprintf("%v/contact", configData.App.FRONTEND_URL)
	policyUrl := fmt.Sprintf("%v/policy", configData.App.FRONTEND_URL)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}


	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]interface{}{
		"firstname": notificationData.Email,
		"contact_us_url": contactUrl,
		"privacy_policy_url": policyUrl,
	})
	if err != nil {
		return fmt.Errorf("error converting data to map, %v", err)
	}

	err = send.SendEmail(n.ExtReq, notificationData.Email, subject, templateFileName, baseTemplateFileName, data)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, ", "))
	}
	return nil
}
