package seed

import (
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedAICreditPackages(logger *utility.Logger, db *gorm.DB) {
	packages := []models.AICreditPackage{
		{
			ID:          utility.GenerateUUID(),
			Name:        "Starter Pack",
			Description: "Perfect for individuals trying AI",
			Price:       15,
			Credits:     1000,
			Benefits:    pq.StringArray{"No expiration date", "Access to GPT-4o models", "Standard API priority", "Email support"},
		},
		{
			ID:          utility.GenerateUUID(),
			Name:        "Pro Bundle",
			Description: "Ideal for power users & researchers",
			Price:       50,
			Credits:     5000,
			Benefits:    pq.StringArray{"No expiration date", "Access to all model tiers", "High priority API access", "Priority chat support", "Shared team pool capable"},
		},
		{
			ID:          utility.GenerateUUID(),
			Name:        "Enterprise Pack",
			Description: "Built credits for organisation",
			Price:       200,
			Credits:     25000,
			Benefits:    pq.StringArray{"Volume discount applied", "Advanced usage analytics", "Dedicated account Manager", "Custom SLA"},
		},
	}

	for _, pkg := range packages {
		var existingPkg models.AICreditPackage
		if err := db.Unscoped().Where("name = ?", pkg.Name).First(&existingPkg).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&pkg).Error; err != nil {
					logger.Error("failed to create AI credit package: " + err.Error())
				}
			} else {
				logger.Error("failed to query AI credit package: " + err.Error())
			}
		} else {
			pkg.ID = existingPkg.ID
			pkg.DeletedAt = gorm.DeletedAt{Valid: false}
			if err := db.Unscoped().Model(&existingPkg).Updates(pkg).Error; err != nil {
				logger.Error("failed to update AI credit package: " + err.Error())
			}
		}
	}
}
