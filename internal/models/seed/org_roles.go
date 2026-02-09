package seed

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedRolesAndPermissions(logger *utility.Logger, db *gorm.DB) {
	roles := []models.OrgRole{
		{
			Name:        "Administrator",
			Description: "Full access, control",
			IsDefault:   true,
		},
		{
			Name:        "Guest",
			Description: "Read-only access",
			IsDefault:   true,
		},
		{
			Name:        "User",
			Description: "Read, write, update",
			IsDefault:   true,
		},
		{
			Name:        "Manager",
			Description: "Read, write, approve",
			IsDefault:   true,
		},
		{
			Name:        "Project Lead",
			Description: "Manage, coordinate, oversee",
			IsDefault:   true,
		},
	}

	for _, role := range roles {
		var existingRole models.OrgRole
		if err := db.Where("name = ?", role.Name).First(&existingRole).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				role.ID = utility.GenerateUUID()
				if err := db.Create(&role).Error; err != nil {
					logger.Error(fmt.Sprintf("Failed to seed role: %s, error: %v", role.Name, err))
				} else {
					// Seed permissions for this new role
					seedPermissionsForRole(logger, db, role)
				}
			} else {
				logger.Error(fmt.Sprintf("Error checking for role %s: %v", role.Name, err))
			}
		}
	}
}

func seedPermissionsForRole(logger *utility.Logger, db *gorm.DB, role models.OrgRole) {
	permission := models.Permission{
		ID:        utility.GenerateUUID(),
		RoleID:    role.ID,
		IsDefault: true,
	}

	switch role.Name {
	case "Administrator":
		permission.PermissionList = models.PermissionList{
			CanRemovePeopleFromOrganization: true,
			CanInviteMembers:                true,
			CanCreateCustomRole:             true,
			CanCreateChannel:                true,
			CanCommentOnThreads:             true,
			CanViewBilling:                  true,
			CanCreateWebhooks:               true,
			CanViewChannels:                 true,
			CanChangeUserOrgRole:            true,
		}
	case "Guest":
		permission.PermissionList = models.PermissionList{
			CanRemovePeopleFromOrganization: false,
			CanInviteMembers:                false,
			CanCreateCustomRole:             false,
			CanCreateChannel:                false,
			CanCommentOnThreads:             false,
			CanViewBilling:                  false,
			CanCreateWebhooks:               false,
			CanViewChannels:                 true,
			CanChangeUserOrgRole:            false,
		}
	case "User":
		permission.PermissionList = models.PermissionList{
			CanRemovePeopleFromOrganization: false,
			CanInviteMembers:                true,
			CanCreateCustomRole:             false,
			CanCreateChannel:                true,
			CanCommentOnThreads:             true,
			CanViewBilling:                  false,
			CanCreateWebhooks:               false,
			CanViewChannels:                 true,
			CanChangeUserOrgRole:            false,
		}
	case "Manager":
		permission.PermissionList = models.PermissionList{
			CanRemovePeopleFromOrganization: true,
			CanInviteMembers:                true,
			CanCreateCustomRole:             true,
			CanCreateChannel:                true,
			CanCommentOnThreads:             true,
			CanViewBilling:                  true,
			CanCreateWebhooks:               true,
			CanViewChannels:                 true,
			CanChangeUserOrgRole:            true,
		}
	case "Project Lead":
		permission.PermissionList = models.PermissionList{
			CanRemovePeopleFromOrganization: true,
			CanInviteMembers:                true,
			CanCreateCustomRole:             true,
			CanCreateChannel:                true,
			CanCommentOnThreads:             true,
			CanViewBilling:                  false,
			CanCreateWebhooks:               false,
			CanViewChannels:                 true,
			CanChangeUserOrgRole:            true,
		}
	}

	if err := db.Create(&permission).Error; err != nil {
		logger.Error(fmt.Sprintf("Failed to seed permission for role %s: %v", role.Name, err))
	}
}
