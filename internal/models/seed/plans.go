package seed

import (
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"github.com/lib/pq"
)

func SeedPlans(logger *utility.Logger, db *gorm.DB) {
	plans := []models.Plan{
		{
			ID:                       utility.GenerateUUID(),
			Name:                     "Free",
			Description:              "Perfect for individuals",
			Benefits:                 pq.StringArray{"Create your own AI Co Workers", "Unlimited AI Co Workers", "AI Credits purchaeable", "1 hour maximum call duration", "Up to 5 buzz participants", "Up to 5 active calls per workspace"},
			Fee:                      0,
			MaxChannels:              -1,
			MaxUsers:                 100,
			CanUpgradeNotifications:  true,
			CanAddUnlimitedChannels:  true,
			CanAddUnlimitedUsers:     false,
			IsForIndividuals:         true,
			IsForSmallBusiness:       false,
			IsForLargeEnterprise:     false,
			Credits:                  10,
			UnlimitedAICoWorkers:     true,
			CreateYourOwnAICoWorkers: true,
			AICreditsPurchasable:     true,
			MaxCallDuration:          60,
			MaxBuzzParticipants:      5,
			MaxActiveCalls:           5,
		},
		{
			ID:                       utility.GenerateUUID(),
			Name:                     "Pro",
			Description:              "Ideal for growing learners",
			Benefits:                 pq.StringArray{"Create your own AI Co Workers", "Unlimited AI Co Workers", "AI Credits purchaeable", "1 hour maximum call duration", "Up to 50 buzz participants", "Up to 50 active calls per workspace", "Advanced controls for administrator only"},
			Fee:                      20,
			MaxChannels:              -1,
			MaxUsers:                 -1,
			CanUpgradeNotifications:  true,
			CanAddUnlimitedChannels:  true,
			CanAddUnlimitedUsers:     true,
			IsForIndividuals:         true,
			IsForSmallBusiness:       true,
			IsForLargeEnterprise:     false,
			Credits:                  50,
			UnlimitedAICoWorkers:     true,
			CreateYourOwnAICoWorkers: true,
			AICreditsPurchasable:     true,
			MaxCallDuration:          60,
			MaxBuzzParticipants:      50,
			MaxActiveCalls:           50,
			AdvancedControls:         true,
		},
		{
			ID:                       utility.GenerateUUID(),
			Name:                     "Pro Plus",
			Description:              "Built for advanced learning teams",
			Benefits:                 pq.StringArray{"Create your own AI Co Workers", "Unlimited AI Co Workers", "AI Credits purchaeable", "1 hour maximum call duration", "Up to 500 buzz partiipants", "Up to 500 active calls per workspace", "Advanced controls for administrator only", "Call records available"},
			Fee:                      100,
			MaxChannels:              -1,
			MaxUsers:                 -1,
			CanUpgradeNotifications:  true,
			CanAddUnlimitedChannels:  true,
			CanAddUnlimitedUsers:     true,
			IsForIndividuals:         false,
			IsForSmallBusiness:       true,
			IsForLargeEnterprise:     false,
			Credits:                  200,
			UnlimitedAICoWorkers:     true,
			CreateYourOwnAICoWorkers: true,
			AICreditsPurchasable:     true,
			MaxCallDuration:          60,
			MaxBuzzParticipants:      500,
			MaxActiveCalls:           500,
			AdvancedControls:         true,
			CallRecordsAvailable:     true,
		},
		{
			ID:                       utility.GenerateUUID(),
			Name:                     "Business",
			Description:              "Designed for organizations",
			Benefits:                 pq.StringArray{"Create your own AI Co Workers", "Unlimited AI Co Workers", "AI Credits purchaeable", "Unlimited call duration", "Up to 1500 buzz participants", "Up to 1500 active calls per workspace", "Call records and transcript available", "Advanced controls for administrator only"},
			Fee:                      150,
			MaxChannels:              -1,
			MaxUsers:                 500,
			MaxNotifications:         -1,
			CanUpgradeNotifications:  true,
			CanAddUnlimitedChannels:  true,
			CanAddUnlimitedUsers:     true,
			IsForIndividuals:         false,
			IsForSmallBusiness:       true,
			IsForLargeEnterprise:     true,
			Credits:                  500,
			UnlimitedAICoWorkers:     true,
			CreateYourOwnAICoWorkers: true,
			AICreditsPurchasable:     true,
			MaxCallDuration:          0,
			MaxBuzzParticipants:      1500,
			MaxActiveCalls:           1500,
			CallRecordsAvailable:     true,
			TranscriptAvailable:      true,
			AdvancedControls:         true,
		},
		{
			ID:                       utility.GenerateUUID(),
			Name:                     "Enterprise",
			Description:              "Tailored for large institutions",
			Benefits:                 pq.StringArray{"Create your own AI Co Workers", "Unlimited AI Co Workers", "AI Credits purchaeable", "Unlimited call duration", "Unlimited buzz participants", "Up to 1000 active calls per workspace", "Call records and transcript available", "Advanced controls for every user"},
			Fee:                      200,
			MaxChannels:              -1,
			MaxUsers:                 -1,
			MaxNotifications:         -1,
			CanUpgradeNotifications:  true,
			CanAddUnlimitedChannels:  true,
			CanAddUnlimitedUsers:     true,
			IsForIndividuals:         false,
			IsForSmallBusiness:       false,
			IsForLargeEnterprise:     true,
			Credits:                  10000,
			UnlimitedAICoWorkers:     true,
			CreateYourOwnAICoWorkers: true,
			AICreditsPurchasable:     true,
			MaxCallDuration:          0,
			MaxBuzzParticipants:      -1,
			MaxActiveCalls:           1000,
			CallRecordsAvailable:     true,
			TranscriptAvailable:      true,
			AdvancedControlsUser:     true,
		},
	}

	for _, plan := range plans {
		var existingPlan models.Plan
		if err := db.Unscoped().Where("name = ?", plan.Name).First(&existingPlan).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&plan).Error; err != nil {
					logger.Error("failed to create plan: " + err.Error())
				}
			} else {
				logger.Error("failed to query plan: " + err.Error())
			}
		} else {
			plan.ID = existingPlan.ID
			plan.DeletedAt = gorm.DeletedAt{Valid: false}

			if err := db.Unscoped().Model(&existingPlan).Updates(plan).Error; err != nil {
				logger.Error("failed to update plan: " + err.Error())
			}
		}

		logger.Info("Plans seeded successfully>>>>>>>>>>>>>>")
	}
}
