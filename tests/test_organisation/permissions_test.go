package test_organisation

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestMasterSystemPermissions(t *testing.T) {
	permissions := models.GetMasterSystemPermissions()
	if len(permissions) != 25 {
		t.Fatalf("expected 25 master permissions, got %d", len(permissions))
	}

	foundManageChannels := false
	for _, p := range permissions {
		if p.Key == models.PermManageChannels {
			foundManageChannels = true
			if p.Category != "channels" {
				t.Errorf("expected category 'channels', got '%s'", p.Category)
			}
		}
	}

	if !foundManageChannels {
		t.Errorf("expected to find can_manage_channels permission in master list")
	}
}

func TestOrgUserHasPermission(t *testing.T) {
	pl := models.PermissionList{
		CanManageChannels: true,
		CanViewChannels:   true,
		CanManageBilling:  false,
	}

	if !models.OrgUserHasPermission(pl, models.PermManageChannels) {
		t.Errorf("expected OrgUserHasPermission to return true for can_manage_channels")
	}

	if !models.OrgUserHasPermission(pl, models.PermViewChannels) {
		t.Errorf("expected OrgUserHasPermission to return true for can_view_channels")
	}

	if models.OrgUserHasPermission(pl, models.PermManageBilling) {
		t.Errorf("expected OrgUserHasPermission to return false for can_manage_billing")
	}
}

func TestPermissionListLegacyBackwardCompatibility(t *testing.T) {
	pl := models.PermissionList{
		CanCreateChannel: true,
	}

	if !models.OrgUserHasPermission(pl, models.PermCreateChannels) {
		t.Errorf("expected legacy CanCreateChannel to map to can_create_channels permission")
	}
}
