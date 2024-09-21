package seed

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedIntegrations(logger *utility.Logger, db *gorm.DB) {
	var count int64

	// Check if integrations already exist in the database
	if err := db.Model(&models.Integrations{}).Where("name IN ?", []string{"Slack", "Microsoft", "Jira Cloud", "Dropbox"}).Count(&count).Error; err != nil {
		logger.Error("Integration seeding: " + err.Error())
		return
	}

	if count > 0 {
		logger.Error("Integrations already exist, skipping seeding...")
		return
	} else {

		slackConf := config.Config.Slack
		perm := "incoming-webhook%20chat%3Awrite%20channels%3Aread%20groups%3Aread"
		authUrl := fmt.Sprintf("'https://slack.com/oauth/v2/authorize?client_id=${%s}&scope=${%s}&redirect_uri=${%s}'", slackConf.ClientId, perm, slackConf.RedirectURI)

		integrations := []models.Integrations{
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Slack",
				AppUrl:              "https://slack.com/api/",
				AuthUrl:             authUrl,
				AppDescription:      "Slack is a cloud-based team business and communication platform.",
				AppLogo:             "https://a.slack-edge.com/fd21de4/marketing/img/nav/logo.svg",
				IsSystemIntegration: true,
			},
		}

		for _, integration := range integrations {
			if err := db.Create(&integration).Error; err != nil {
				logger.Error("failed to seed integration: " + err.Error())
			}
		}

		logger.Info("System integrations seeded successfully.")
	}
}
