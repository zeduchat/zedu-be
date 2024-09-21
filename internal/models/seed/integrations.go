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
				JSONUrl:             "https://systems.telex.im/slack",
				JSONSchema: map[string]interface{}{
					
				},
				AuthCredential:      "slack-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Microsoft",
				JSONUrl:             "https://systems.telex.im/microsoft",
				AuthCredential:      "microsoft-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Jira Cloud",
				JSONUrl:             "https://systems.telex.im/jira",
				AuthCredential:      "jira-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "Dropbox",
				JSONUrl:             "https://systems.telex.im/dropbox",
				AuthCredential:      "dropbox-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "github",
				JSONUrl:             "https://systems.telex.im/github",
				AuthCredential:      "github-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "domino",
				JSONUrl:             "https://api.dominodatalab.com/v1/",
				AuthCredential:      "domino-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "kuda",
				JSONUrl:             "https://systems.telex.im/kuda",
				AuthCredential:      "kuda-auth-token",
				IsSystemIntegration: true,
				CreatedAt:           time.Now(),
			},
			{
				ID:                  utility.GenerateUUID(),
				Name:                "moniepoint",
				JSONUrl:             "https://systems.telex.im/moniepoint",
				AuthCredential:      "moniepoint-auth-token",
				IsSystemIntegration: true,

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
