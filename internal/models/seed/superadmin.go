package seed

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

// SeedSuperAdmin seeds the superadmin from environment variables.
// The superadmin must be an existing Telex user (validated by email).
// This function is idempotent - it skips if the admin already exists.
func SeedSuperAdmin(logger *utility.Logger, db *gorm.DB) {
	cfg := config.GetConfig().Admin

	// Skip if env vars not configured
	if cfg.SUPER_ADMIN_EMAIL == "" {
		logger.Info("SUPER_ADMIN_EMAIL not set, skipping superadmin seeding")
		return
	}

	email := strings.ToLower(cfg.SUPER_ADMIN_EMAIL)

	// Check if admin already exists
	var existingAdmin models.Admin
	if err := db.Where("email = ?", email).First(&existingAdmin).Error; err == nil {
		logger.Info(fmt.Sprintf("Superadmin with email %s already exists, skipping seeding", email))
		return
	}

	// Validate that email belongs to an existing Telex user
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		logger.Error(fmt.Sprintf("Superadmin seeding failed: email %s is not a registered Telex user", email))
		fmt.Printf("ERROR: Superadmin seeding failed: email %s is not a registered Telex user\n", email)
		return
	}

	// Validate password is set
	if cfg.SUPER_ADMIN_PASSWORD == "" {
		logger.Error("SUPER_ADMIN_PASSWORD not set, skipping superadmin seeding")
		return
	}

	// Hash password
	hashedPassword, err := utility.HashPassword(cfg.SUPER_ADMIN_PASSWORD)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to hash superadmin password: %v", err))
		return
	}

	// Determine name (from env or user profile)
	name := cfg.SUPER_ADMIN_NAME
	if name == "" {
		name = user.Name
	}
	if name == "" {
		name = "Super Admin"
	}

	// Create superadmin
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
