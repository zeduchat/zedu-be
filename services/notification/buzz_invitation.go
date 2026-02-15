package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendBuzzInvitationEmail() error {
	var (
		notificationData     = models.SendBuzzInvitationEmail{}
		templateFileName     = "buzz_invitation.html"
		baseTemplateFileName = ""
		errs                 []string
		configData           = config.GetConfig()
	)
	subject := "Subject: You've been invited to a Buzz on Zedu"
	contactUrl := fmt.Sprintf("%v/contact", configData.App.FRONTEND_URL)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]any{
		"invitee_name":   notificationData.InviteeName,
		"inviter_name":   notificationData.InviterName,
		"buzz_code":      notificationData.BuzzCode,
		"join_link":      notificationData.JoinLink,
		"contact_us_url": contactUrl,
	})
	if err != nil {
		return fmt.Errorf("error converting data to map, %v", err)
	}

	err = send.SendEmail(n.ExtReq, notificationData.Email, subject, templateFileName, baseTemplateFileName, data)
	if err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("%v", strings.Join(errs, ", "))
	}
	return nil
}
