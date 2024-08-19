package seed

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedRolesAndPermissions(logger *utility.Logger, db *gorm.DB) {
	var count int64
	if err := db.Model(&models.OrgRole{}).Where("name IN ?", []string{"Project Lead", "Manager"}).Count(&count).Error; err != nil {
		fmt.Println(err)
		logger.Error("org role seeding: " + err.Error())
	}

	if count > 0 {
		logger.Error("Roles 'Project Lead' or 'Manager' already exist, skipping seeding...")
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
					CanRemovePeopleFromOrganization: true,
					CanInviteMembers:                true,
					CanCreateCustomRole:             true,
					CanCreateChannel:                true,
					CanCommentOnThreads:             true,
					CanViewBilling:                  true,
					CanCreateWebhooks:               true,
					CanViewChannels:                 true,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[1].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanRemovePeopleFromOrganization: false,
					CanInviteMembers:                false,
					CanCreateCustomRole:             false,
					CanCreateChannel:                false,
					CanCommentOnThreads:             false,
					CanViewBilling:                  false,
					CanCreateWebhooks:               false,
					CanViewChannels:                 true,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[2].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanRemovePeopleFromOrganization: false,
					CanInviteMembers:                true,
					CanCreateCustomRole:             false,
					CanCreateChannel:                true,
					CanCommentOnThreads:             true,
					CanViewBilling:                  false,
					CanCreateWebhooks:               false,
					CanViewChannels:                 true,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[3].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanRemovePeopleFromOrganization: true,
					CanInviteMembers:                true,
					CanCreateCustomRole:             true,
					CanCreateChannel:                true,
					CanCommentOnThreads:             true,
					CanViewBilling:                  true,
					CanCreateWebhooks:               true,
					CanViewChannels:                 true,
				},
			},
			{
				ID:        utility.GenerateUUID(),
				RoleID:    roles[4].ID,
				IsDefault: true,
				PermissionList: models.PermissionList{
					CanRemovePeopleFromOrganization: true,
					CanInviteMembers:                true,
					CanCreateCustomRole:             true,
					CanCreateChannel:                true,
					CanCommentOnThreads:             true,
					CanViewBilling:                  false,
					CanCreateWebhooks:               false,
					CanViewChannels:                 true,
				},
			},
		}

		for _, role := range roles {
			if err := db.Create(&role).Error; err != nil {
				logger.Error(fmt.Sprintf("Failed to seed role: %s, error: %v", role.Name, err))
				fmt.Printf("Failed to seed role: %s, error: %v", role.Name, err)
			}
		}

		for _, permission := range permissions {
			if err := db.Create(&permission).Error; err != nil {
				logger.Error(fmt.Sprintf("Failed to seed permission for role: %s, error: %v", permission.RoleID, err))
				fmt.Printf("Failed to seed permission for role: %s, error: %v", permission.RoleID, err)
			}
		}
	}
}
