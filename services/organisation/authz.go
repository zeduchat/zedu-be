package organisation

import (
	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

// userCanOrOwner returns true if the user is the org owner OR has the given
// permission via their org role. This replaces owner-only checks with a
// permission-first approach while preserving the owner bypass.
func userCanOrOwner(db *gorm.DB, userID, orgID, permission string) bool {
	// Owner always passes
	var org models.Organisation
	if isOwner, _ := org.IsOwnerOfOrganisation(db, userID, orgID); isOwner {
		return true
	}

	// Check the user's role permission in this org
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

	return models.OrgUserHasPermission(role.Permissions.PermissionList, permission)
}
