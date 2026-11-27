package test_profile

import (
	"fmt"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

// Ziki #2347: a user can have at most one base profile (organisation_id IS NULL).
// The (userid, organisation_id) index treats NULLs as distinct, so this needs
// the partial unique index idx_profiles_user_base.
func TestProfilesRejectSecondBaseProfile(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	userID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()
	if err := db.Create(&models.User{ID: userID, Name: "Base Unique", Email: fmt.Sprintf("baseunique%s@qa.team", utility.GenerateUUID())}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Where("userid = ?", userID).Delete(&models.Profile{})
		db.Unscoped().Where("id = ?", userID).Delete(&models.User{})
	})

	newProfile := func(org *string) models.Profile {
		now := time.Now()
		return models.Profile{ID: utility.GenerateUUID(), Userid: userID, OrganisationID: org, FirstName: "Base", FullName: "Base", CreatedAt: now, UpdatedAt: now}
	}

	first := newProfile(nil)
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("first base profile should be created: %v", err)
	}

	second := newProfile(nil)
	if err := db.Create(&second).Error; err == nil {
		t.Fatalf("expected a second base profile for the same user to be rejected")
	}

	orgProfile := newProfile(&orgID)
	if err := db.Create(&orgProfile).Error; err != nil {
		t.Fatalf("an org-scoped profile next to the base profile must still be allowed: %v", err)
	}

	var count int64
	db.Model(&models.Profile{}).Where("userid = ? AND organisation_id IS NULL", userID).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 base profile, got %d", count)
	}
}
