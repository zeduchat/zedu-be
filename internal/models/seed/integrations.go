package seed

import (
	"gorm.io/gorm"

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

		// slackConf := config.Config.Slack
		// perm := "incoming-webhook%20chat%3Awrite%20channels%3Aread%20groups%3Aread"
		// authUrl := fmt.Sprintf("https://slack.com/oauth/v2/authorize?client_id=%s&scope=%s&redirect_uri=%s", slackConf.ClientId, perm, slackConf.RedirectURI)

		integrations := []models.Integrations{
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Slack",
				AppUrl:              "https://slack.com",
				JSONUrl:             "https://system-integration.telex.im/slack.json",
				AppDescription:      "Slack is a cloud-based team business and communication platform.",
				AppLogo:             "https://media.tifi.tv/telexbucket/public/logos/slack.png",
				IsSystemIntegration: true,
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "GitHub",
				AppUrl:              "https://github.com",
				JSONUrl:             "https://system-integration.telex.im/github.json",
				AppDescription:      "GitHub is a web-based hosting service for version control using Git.",
				AppLogo:             "https://media.tifi.tv/telexbucket/public/logos/github.png",
				IsSystemIntegration: true,
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Jira",
				AppUrl:              "https://www.atlassian.com/software/jira",
				JSONUrl:             "https://system-integration.telex.im/jira.json",
				AppDescription:      "Jira is a proprietary issue tracking product developed by Atlassian.",
				AppLogo:             "https://media.tifi.tv/telexbucket/public/logos/jira.png",
				IsSystemIntegration: true,
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Microsoft",
				AppUrl:              "https://www.microsoft.com",
				JSONUrl:             "https://system-integration.telex.im/microsoft.json",
				AppDescription:      "Microsoft Corporation is a multinational technology corporation.",
				AppLogo:             "https://media.tifi.tv/telexbucket/public/logos/microsoft.png",
				IsSystemIntegration: true,
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Microsoft Teams",
				AppUrl:              "https://www.microsoft.com/en-us/microsoft-teams/group-chat-software",
				JSONUrl:             "https://system-integration.telex.im/teams.json",
				AppDescription:      "Microsoft Teams is a proprietary business communication platform developed by Microsoft.",
				AppLogo:             "https://media.tifi.tv/telexbucket/public/logos/microsoft-teams.png",
				IsSystemIntegration: true,
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Dropbox",
				AppUrl:              "https://www.dropbox.com",
				JSONUrl:             "https://system-integration.telex.im/dropbox.json",
				AppDescription:      "Dropbox is a file hosting service operated by the American company Dropbox, Inc.",
				AppLogo:             "https://media.tifi.tv/telexbucket/public/logos/dropbox.png",
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
