package seed

import (
	"fmt"
	"log"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SeedRolesAndPermissions(db *gorm.DB) {
	var count int64
	if err := db.Model(&models.OrgRole{}).Where("name IN ?", []string{"Project Lead", "Manager"}).Count(&count).Error; err != nil {
		fmt.Println(err)
	}

	if count > 0 {
		log.Println("Roles 'Project Lead' or 'Manager' already exist, skipping seeding...")
		fmt.Println("Roles 'Project Lead' or 'Manager' already exist, skipping seeding...")
	} else {
		roles := []models.OrgRole{
			{
				ID:          utility.GenerateUUID(),
				Name:        "Administrator",
				Description: "Full access, control",
				IsDefault:   true,
			},
			{
				ID:          utility.GenerateUUID(),
				Name:        "Guest",
				Description: "Read-only access",
				IsDefault:   true,
			},
			{
				ID:          utility.GenerateUUID(),
				Name:        "User",
				Description: "Read, write, update",
				IsDefault:   true,
			},
			{
				ID:          utility.GenerateUUID(),
				Name:        "Manager",
				Description: "Read, write, approve",
				IsDefault:   true,
			},
			{
				ID:          utility.GenerateUUID(),
				Name:        "Project Lead",
				Description: "Manage, coordinate, oversee",
				IsDefault:   true,
			},
		}

		permissions := []models.Permission{
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[0].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanViewTransactions:       true,
					CanViewRefunds:            true,
					CanLogRefund:              true,
					CanViewUser:               true,
					CanEditUser:               true,
					CanCreateUser:             true,
					CanBlacklistWhitelistUser: true,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[1].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanViewTransactions:       true,
					CanViewRefunds:            false,
					CanLogRefund:              false,
					CanViewUser:               true,
					CanEditUser:               false,
					CanCreateUser:             false,
					CanBlacklistWhitelistUser: false,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[2].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanViewTransactions:       true,
					CanViewRefunds:            true,
					CanLogRefund:              false,
					CanViewUser:               true,
					CanEditUser:               false,
					CanCreateUser:             false,
					CanBlacklistWhitelistUser: false,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[3].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanViewTransactions:       true,
					CanViewRefunds:            true,
					CanLogRefund:              true,
					CanViewUser:               true,
					CanEditUser:               true,
					CanCreateUser:             true,
					CanBlacklistWhitelistUser: true,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[4].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanViewTransactions:       true,
					CanViewRefunds:            true,
					CanLogRefund:              true,
					CanViewUser:               true,
					CanEditUser:               true,
					CanCreateUser:             false,
					CanBlacklistWhitelistUser: false,
				},
			},
		}

		for _, role := range roles {
			if err := db.Create(&role).Error; err != nil {
				log.Printf("Failed to seed role: %s, error: %v", role.Name, err)
				fmt.Printf("Failed to seed role: %s, error: %v", role.Name, err)
			}
		}

		for _, permission := range permissions {
			if err := db.Create(&permission).Error; err != nil {
				log.Printf("Failed to seed permission for role: %s, error: %v", permission.RoleID, err)
				fmt.Printf("Failed to seed permission for role: %s, error: %v", permission.RoleID, err)
			}
		}
	}
}
