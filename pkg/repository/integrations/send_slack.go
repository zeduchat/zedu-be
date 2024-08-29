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
		PreText:    fmt.Sprintf("_Telex Channel: %s_", req.PretextChannel),
		AuthorName: req.AuthorName,
		Title:      fmt.Sprintf(":bell: %s", req.TitleEvent),
		TitleLink:  req.TitleLink,
	}

	title := req.TitleAction

	if len(title) > 25 {
		title = fmt.Sprintf("%s...", title[:25])
	}

	attachment.AddField(slack.Field{
		Title: "Action",
		Value: title,
		Short: true,
	})

	attachment.AddField(slack.Field{
		Title: "Status",
		Value: req.StatusValue,
		Short: true,
	})

	attachment.AddAction(slack.Action{
		Type:  "button",
		Text:  "Go to Channel",
		Url:   req.TitleLink,
		Style: "primary",
	})

	msg := slack.Message{
		Text:        fmt.Sprintf("*Telex Organization: %s*", req.OrgName),
		Attachments: []slack.Attachment{attachment},
		Markdown:    true,
	}

	logger.Info("Sending to slack webhook: " + req.WebhookUrl)

	err := slack.Send(req.WebhookUrl, msg)

	if err != nil {
		logger.Error("an error occured while sending slack notifaction: ", err.Error())
		return err
	}

	logger.Info("notifcation successfully sent to slack from channel ", req.PretextChannel)

	return nil
}
