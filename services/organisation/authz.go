package organisation

import (
	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

// isUserOwnerOrOwnerRole returns true if the user is the organization owner
// or holds the default "Owner" role in the organization.
func isUserOwnerOrOwnerRole(db *gorm.DB, userID, orgID string) bool {
	var org models.Organisation
	if isOwner, _ := org.IsOwnerOfOrganisation(db, userID, orgID); isOwner {
		return true
	}

	var oum models.OrgUserManagement
	membership, err := oum.GetByIDs(db, userID, orgID)
	if err != nil || membership.RoleID == "" {
		return false
	}

	var orgRole models.OrgRole
	role, err := orgRole.GetAOrgRoleById(db, membership.RoleID)
	if err != nil {
		return false
	}

	return role.Name == models.OrgRoleNameOwner
}

// UserCanOrOwner reports whether userID is the owner of orgID, holds the
// Owner role there, or holds `permission` through an active org role.
// It fails closed: any lookup error, missing or deactivated membership,
// or empty id returns false. Permission names are in internal/models/permissions.go.
func UserCanOrOwner(db *gorm.DB, userID, orgID, permission string) bool {
	if userID == "" || orgID == "" {
		return false
	}

	var org models.Organisation
	if isOwner, _ := org.IsOwnerOfOrganisation(db, userID, orgID); isOwner {
		return true
	}

	var oum models.OrgUserManagement
	membership, err := oum.GetByIDs(db, userID, orgID)
	if err != nil || membership.IsDeactivated || membership.RoleID == "" {
		return false
	}

	var orgRole models.OrgRole
	role, err := orgRole.GetAOrgRoleById(db, membership.RoleID)
	if err != nil {
		return false
	}

	if role.Name == models.OrgRoleNameOwner {
		return true
	}
	return models.OrgUserHasPermission(role.Permissions.PermissionList, permission)
}

// userCanOrOwner is kept for existing callers in this package.
func userCanOrOwner(db *gorm.DB, userID, orgID, permission string) bool {
	return UserCanOrOwner(db, userID, orgID, permission)
}
