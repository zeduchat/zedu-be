package seed

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SeedPlans(logger *utility.Logger, db *gorm.DB) {
	var count int64

	if err := db.Model(&models.Plan{}).Where("name IN ?", []string{"Starter", "Business", "Enterprise"}).Count(&count).Error; err != nil {
		logger.Error("plan seeding: " + err.Error())
		return
	}

	if count > 0 {
		logger.Error("Plans already exist, skipping seeding...")
		return
	} else {

		plans := []models.Plan{
			{
				ID:                      utility.GenerateUUID(),
				Name:                    "Free",
				Fee:                     0,
				MaxChannels:             -1,
				MaxUsers:                100, 
				CanUpgradeNotifications: true,
				CanAddUnlimitedChannels: true,
				CanAddUnlimitedUsers:    false, // limited to 100 users
				IsForIndividuals:        true,
				IsForSmallBusiness:      false,
				IsForLargeEnterprise:    false,
				Credits:                 0, // No free credits, credits are purchasable
			},
			{
				ID:                      utility.GenerateUUID(),
				Name:                    "Business",
				Fee:                     50,
				MaxChannels:             -1,
				MaxUsers:                500, // Up to 500 human users
				MaxNotifications:        -1,
				CanUpgradeNotifications: true,
				CanAddUnlimitedChannels: true,
				CanAddUnlimitedUsers:    false, // limited to 500 users
				IsForIndividuals:        false,
				IsForSmallBusiness:      true,
				IsForLargeEnterprise:    false,
				Credits:                 100, // 100 free AI credits monthly + more purchasable
			},

			{
				ID:                      utility.GenerateUUID(),
				Name:                    "Starter",
				Fee:                     10,
				MaxChannels:             5,
				MaxUsers:                5,
				MaxNotifications:        250,
				CanUpgradeNotifications: false,
				CanAddUnlimitedChannels: false,
				CanAddUnlimitedUsers:    false,
				IsForIndividuals:        true,
				IsForSmallBusiness:      false,
				IsForLargeEnterprise:    false,
				Credits:                 1000,
			},
			{
				ID:                      utility.GenerateUUID(),
				Name:                    "Enterprise",
				Fee:                     1000,
				MaxChannels:             -1,
				MaxUsers:                -1,
				MaxNotifications:        -1,
				CanUpgradeNotifications: true,
				CanAddUnlimitedChannels: true,
				CanAddUnlimitedUsers:    true,
				IsForIndividuals:        false,
				IsForSmallBusiness:      false,
				IsForLargeEnterprise:    true,
				Credits:                 10000,
			},
		}

		db = db.Debug()

		for _, plan := range plans {
			if err := db.Create(&plan).Error; err != nil {
				logger.Error("failed to seed plan: " + err.Error())
			}
		}
	}
}
