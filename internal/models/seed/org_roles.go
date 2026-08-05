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
			Name:        models.OrgRoleNameOwner,
			Description: "Organisation owner with full administration authority",
			IsDefault:   true,
		},
		{
			Name:        models.OrgRoleNameAdministrator,
			Description: "Full access, control",
			IsDefault:   true,
		},
		{
			Name:        models.OrgRoleNameGuest,
			Description: "Read-only access",
			IsDefault:   true,
		},
		{
			Name:        models.OrgRoleNameUser,
			Description: "Read, write, update",
			IsDefault:   true,
		},
		{
			Name:        models.OrgRoleNameManager,
			Description: "Read, write, approve",
			IsDefault:   true,
		},
		{
			Name:        models.OrgRoleNameProjectLead,
			Description: "Manage, coordinate, oversee",
			IsDefault:   true,
		},
		{
			Name:        models.OrgRoleNameBot,
			Description: "Automated agent or integration identity",
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
	case models.OrgRoleNameOwner:
		permission.PermissionList = models.PermissionList{
			CanManageChannels:     true,
			CanManageMembers:      true,
			CanManageOrganization: true,
			CanManageSettings:     true,
			CanManageBilling:      true,
			CanManageAgents:       true,
			CanManageWorkflows:    true,
			CanManageIntegrations: true,
			CanManageSecurity:     true,
			CanManageRoles:        true,
			CanViewAnalytics:      true,
			CanViewBilling:        true,
			CanViewChannels:       true,
			CanEditMessages:       true,
			CanDeleteMessages:     true,
			CanDeleteFiles:        true,
			CanCreateChannels:     true,
			CanCreateAgents:       true,
			CanCreateRole:         true,
			CanCreateWebhooks:     true,
			CanArchiveChannels:    true,
			CanInviteMembers:      true,
			CanRemovePeople:       true,
			CanCommentThreads:     true,
			CanChangeUserOrgRole:  true,
		}
	case models.OrgRoleNameAdministrator:
		permission.PermissionList = models.PermissionList{
			CanManageChannels:     true,
			CanManageMembers:      true,
			CanManageOrganization: true,
			CanManageSettings:     true,
			CanManageAgents:       true,
			CanManageWorkflows:    true,
			CanManageIntegrations: true,
			CanManageRoles:        true,
			CanViewAnalytics:      true,
			CanViewBilling:        true,
			CanViewChannels:       true,
			CanEditMessages:       true,
			CanDeleteMessages:     true,
			CanDeleteFiles:        true,
			CanCreateChannels:     true,
			CanCreateAgents:       true,
			CanCreateRole:         true,
			CanCreateWebhooks:     true,
			CanArchiveChannels:    true,
			CanInviteMembers:      true,
			CanRemovePeople:       true,
			CanCommentThreads:     true,
			CanChangeUserOrgRole:  true,
		}
	case models.OrgRoleNameManager:
		permission.PermissionList = models.PermissionList{
			CanManageChannels:    true,
			CanManageMembers:     true,
			CanManageSettings:    true,
			CanManageAgents:      true,
			CanViewAnalytics:     true,
			CanViewBilling:       true,
			CanViewChannels:      true,
			CanEditMessages:      true,
			CanDeleteMessages:    true,
			CanCreateChannels:    true,
			CanCreateAgents:      true,
			CanCreateRole:        true,
			CanArchiveChannels:   true,
			CanInviteMembers:     true,
			CanRemovePeople:      true,
			CanCommentThreads:    true,
			CanChangeUserOrgRole: true,
		}
	case models.OrgRoleNameProjectLead:
		permission.PermissionList = models.PermissionList{
			CanManageChannels: true,
			CanViewChannels:   true,
			CanEditMessages:   true,
			CanDeleteMessages: true,
			CanCreateChannels: true,
			CanInviteMembers:  true,
			CanCommentThreads: true,
		}
	case models.OrgRoleNameUser:
		permission.PermissionList = models.GetUserDefaultPermissions()
	case models.OrgRoleNameBot:
		permission.PermissionList = models.PermissionList{
			CanViewAnalytics: true,
			CanViewChannels:  true,
		}
	case models.OrgRoleNameGuest:
		permission.PermissionList = models.PermissionList{
			CanViewChannels: true,
		}
	}

	if err := db.Create(&permission).Error; err != nil {
		logger.Error(fmt.Sprintf("Failed to seed permission for role %s: %v", role.Name, err))
	}
}
