package seed

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedSuperAdmin(logger *utility.Logger, db *gorm.DB) {
	cfg := config.GetConfig().Admin

	if cfg.SUPER_ADMIN_EMAIL == "" {
		logger.Info("SUPER_ADMIN_EMAIL not set, skipping superadmin seeding")
		return
	}

	email := strings.ToLower(cfg.SUPER_ADMIN_EMAIL)

	var existingAdmin models.Admin
	if err := db.Where("email = ?", email).First(&existingAdmin).Error; err == nil {
		logger.Info(fmt.Sprintf("Superadmin with email %s already exists, skipping seeding", email))
		return
	}

	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		logger.Error(fmt.Sprintf("Superadmin seeding failed: email %s is not a registered Telex user", email))
		fmt.Printf("ERROR: Superadmin seeding failed: email %s is not a registered Telex user\n", email)
		return
	}

	if cfg.SUPER_ADMIN_PASSWORD == "" {
		logger.Error("SUPER_ADMIN_PASSWORD not set, skipping superadmin seeding")
		return
	}

	hashedPassword, err := utility.HashPassword(cfg.SUPER_ADMIN_PASSWORD)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to hash superadmin password: %v", err))
		return
	}

	name := cfg.SUPER_ADMIN_NAME
	if name == "" {
		name = user.Name
	}
	if name == "" {
		name = "Super Admin"
	}

	admin := models.Admin{
		ID:       utility.GenerateUUID(),
		Email:    email,
		Name:     name,
		Password: hashedPassword,
		Role:     models.RoleSuperAdmin,
		IsActive: true,
	}

	if err := db.Create(&admin).Error; err != nil {
		logger.Error(fmt.Sprintf("Failed to seed superadmin: %v", err))
		fmt.Printf("ERROR: Failed to seed superadmin: %v\n", err)
		return
	}

	logger.Info(fmt.Sprintf("Successfully seeded superadmin: %s", email))
	fmt.Printf("Successfully seeded superadmin: %s\n", email)
}
