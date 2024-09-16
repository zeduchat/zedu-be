package seed

import (
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SeedIntegrations(logger *utility.Logger, db *gorm.DB) {
	var count int64

	// Check if integrations already exist in the database
	if err := db.Model(&models.Integrations{}).Where("name IN ?",[]string{"Slack", "Microsoft", "Jira Cloud", "Dropbox"}).Count(&count).Error; err != nil {
		logger.Error("Integration seeding: " + err.Error())
		return
	}

	if count > 0 {
		logger.Error("Integrations already exist, skipping seeding...")
		return
	} else {
		// Define system integrations to be seeded
		integrations := []models.Integrations{
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Slack",
				JSONUrl:             "https://slack.com/api/",
				AuthCredential:      "slack-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Microsoft",
				JSONUrl:             "https://graph.microsoft.com/v1.0/",
				AuthCredential:      "microsoft-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Jira Cloud",
				JSONUrl:             "https://your-domain.atlassian.net/rest/api/3/",
				AuthCredential:      "jira-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Dropbox",
				JSONUrl:             "https://api.dropboxapi.com/2/",
				AuthCredential:      "dropbox-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Microsoft Teams",
				JSONUrl:             "https://graph.microsoft.com/v1.0/teams/",
				AuthCredential:      "teams-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
		}

		// Seed the integrations into the database
		for _, integration := range integrations {
			if err := db.Create(&integration).Error; err != nil {
				logger.Error("failed to seed integration: " + err.Error())
			}
		}

		logger.Info("System integrations seeded successfully.")
	}
}
