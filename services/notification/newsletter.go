package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/send"
)

func (n NotificationObject) SendNewsletterMail() error {
	var (
		notificationData     = models.SendNewsletterSubscriptionMail{}
		subject              = "Subject: Welcome to Our Newsletter! Your Subscription is Confirmed😃"
		templateFileName     = "newsletter.html"
		baseTemplateFileName = ""
		configData           = config.GetConfig()
	)

	err := json.Unmarshal([]byte(n.Notification.Data), &notificationData)
	if err != nil {
		return fmt.Errorf("error decoding saved notification data, %v", err)
	}

	blogUrl := fmt.Sprintf("%v/blogs", configData.App.Url)
	contactUrl := fmt.Sprintf("%v/contact", configData.App.Url)

	data, err := ConvertToMapAndAddExtraData(notificationData, map[string]interface{}{
		"firstname":   thisOrThatStr(notificationData.Email, "there!"),
		"blog_url":    blogUrl,
		"contact_url": contactUrl,
	})

	if err != nil {
		return fmt.Errorf("error converting data to map, %v, %v", err, strings.Join([]string{err.Error()}, ", "))
	}

	return send.SendEmail(n.ExtReq, notificationData.Email, subject, templateFileName, baseTemplateFileName, data)
}
