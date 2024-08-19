package integrations

import (
	"fmt"

	"github.com/anthonycorbacho/slack-webhook"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SendSlacKNotification(req models.SendSlackRequest, logger *utility.Logger) error {

	attachment := slack.Attachment{
		Fallback:   "",
		Color:      req.Color,
		PreText:    fmt.Sprintf("_%s_", req.PretextChannel),
		AuthorName: req.AuthorName,
		Title:      req.TitleEvent,
		TitleLink:  req.TitleLink,
	}

	attachment.AddField(slack.Field{
		Title: req.TitleAction,
		Value: req.StatusValue,
		Short: false,
	})
	attachment.AddAction(slack.Action{
		Type:  "button",
		Text:  "Go to Channel",
		Url:   req.TitleLink,
		Style: "primary",
	})

	msg := slack.Message{
		Text:        fmt.Sprintf("*%s*", req.OrgName),
		Attachments: []slack.Attachment{attachment},
		Markdown:    true,
	}

	err := slack.Send(req.WebhookUrl, msg)

	if err != nil {
		logger.Error("an error occured while sending slack notifaction: ", err.Error())
		return err
	}

	logger.Info("notifcation successfully sent to slack from channel ", req.PretextChannel)

	return nil
}
