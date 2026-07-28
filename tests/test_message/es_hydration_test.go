package test_message

import (
	"fmt"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests/Test_profile"
	"github.com/hngprojects/telex_be/utility"
)

func TestESMessageAndThreadHydration(t *testing.T) {
	_, profileController := test_profile.SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	uID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()

	testUser := models.User{
		ID:    uID,
		Name:  "ES Hydration User",
		Email: fmt.Sprintf("eshydration%s@qa.team", utility.GenerateUUID()),
	}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	var profModel models.Profile
	prof, err := profModel.GetOrCreateProfileForOrg(db, uID, orgID)
	if err != nil {
		t.Fatalf("Failed to create org profile: %v", err)
	}

	t.Run("Hydrate Message Documents Test", func(t *testing.T) {
		msgDocLegacy := models.MessageDocument{
			ID:             utility.GenerateUUID(),
			UserID:         uID,
			OrganisationID: orgID,
			Content:        "Legacy ES Message",
			ProfileID:      "", // Legacy missing ProfileID
		}

		hydrated := models.HydrateMessageProfiles(db, []models.MessageDocument{msgDocLegacy})
		if len(hydrated) == 0 {
			t.Fatalf("Expected non-empty hydrated slice")
		}
		if hydrated[0].ProfileID != prof.ID {
			t.Errorf("Expected hydrated ProfileID to match org profile %s, got %s", prof.ID, hydrated[0].ProfileID)
		}
	})

	t.Run("Hydrate Thread Documents Test", func(t *testing.T) {
		threadDocLegacy := models.ThreadDocument{
			ID:             utility.GenerateUUID(),
			UserId:         uID,
			OrganisationID: orgID,
			Content:        "Legacy ES Thread",
			ProfileID:      "", // Legacy missing ProfileID
		}

		hydrated := models.HydrateThreadProfiles(db, []models.ThreadDocument{threadDocLegacy})
		if len(hydrated) == 0 {
			t.Fatalf("Expected non-empty hydrated slice")
		}
		if hydrated[0].ProfileID != prof.ID {
			t.Errorf("Expected hydrated ProfileID to match org profile %s, got %s", prof.ID, hydrated[0].ProfileID)
		}
	})
}
