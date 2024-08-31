package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendInvitationLink() error {
	var (
		notificationData     = models.SendInvitationLink{}
		templateFileName     = "send_invitation.html"
		baseTemplateFileName = ""
		errs                 []string
		user                 models.User
	)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	subject := "Subject: Secure Login: Your Invitation..."

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]interface{}{"firstname": thisOrThatStr(user.Profile.FirstName, user.Email)})
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
