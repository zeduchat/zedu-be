package integrations

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

var ColorMapping = map[string]string{
	"success": "#008000",
	"error":   "#800000",
}

func BuildSlackRequest(feed models.FeedWebHookRequest, db *gorm.DB, logger *utility.Logger) error {
	var (
		channel    models.Channels
		slackentry models.SlackTelex
		org        models.Organisation
		slackReq   models.SendSlackRequest
		chanReq   models.ChannelInfo
	)

	chanReq.ChannelID = feed.ChannelID

	chanresp, err := channel.GetChannelsByID(db, chanReq)

	if err != nil {
		return errors.New("failed to fetch channel")
	}

	err = slackentry.GetSlackWebhookUrl(db, chanresp.OrganisationID)

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	if err != nil {
		return errors.New("failed to fetch slack integration")
	}

	org, err = org.GetOrgByID(db, chanresp.OrganisationID)

	if err != nil {
		return errors.New("failed to fetch channel organisation")
	}

	slackReq = models.SendSlackRequest{
		AuthorName:     feed.UserName,
		PretextChannel: chanresp.Name,
		TitleEvent:     feed.EventName,
		TitleAction:    feed.ActionType,
		TitleLink:      fmt.Sprintf("%s/dashboard/channels/%s", config.Config.App.FRONTEND_URL, feed.ChannelID),
		OrgName:        org.Name,
		StatusValue:    feed.Status,
		WebhookUrl:     slackentry.URL,
		Color:          ColorMapping[strings.ToLower(feed.Status)],
	}

	logger.Info("sending notification to slack channel %s", slackentry.Channel)

	err = SendSlacKNotification(slackReq, logger)

	if err != nil {
		logger.Error("sending notification to slack failed: " + err.Error())
		return err
	}

	return nil
}
