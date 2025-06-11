package seed

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SeedCreditPackages(logger *utility.Logger, db *gorm.DB) {
	var count int64
	if err := db.Model(&models.CreditPackage{}).Count(&count).Error; err != nil {
		logger.Error("credit package seeding: " + err.Error())
		return
	}

	if count > 0 {
		logger.Info("Credit packages already exist, skipping seeding...")
		return
	}

	packages := []models.CreditPackage{
		{
			ID:       utility.GenerateUUID(),
			Name:     "Starter Pack",
			Credits:  1000,
			Price:    5.00,
			Currency: "USD",
		},
		{
			ID:       utility.GenerateUUID(),
			Name:     "Pro Bundle",
			Credits:  5000,
			Price:    20.00,
			Currency: "USD",
		},
		{
			ID:       utility.GenerateUUID(),
			Name:     "Enterprise Pack",
			Credits:  10000,
			Price:    35.00,
			Currency: "USD",
		},
	}

	for _, pack := range packages {
		if err := db.Create(&pack).Error; err != nil {
			logger.Error("failed to seed credit package: " + err.Error())
		}
	}

	logger.Info("Credit packages seeded successfully")
}
