package test_subscription

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/subscription"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetSubscriptionPlans_Fields(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	if err := db.Postgresql.AutoMigrate(&models.Plan{}); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	seed.SeedPlans(logger, db.Postgresql)

	plans, _, err := subscription.GetSubscriptionPlans(db.Postgresql)

	assert.NoError(t, err)
	assert.NotNil(t, plans)
	assert.Greater(t, len(*plans), 0)

	for _, plan := range *plans {
		assert.NotEmpty(t, plan.ID)
		assert.NotEmpty(t, plan.Name)
		assert.NotEmpty(t, plan.Description)
		assert.NotEmpty(t, plan.Benefits)
		assert.NotZero(t, plan.CreatedAt)
		assert.NotZero(t, plan.UpdatedAt)
	}
}

func TestGetSubscriptions_NoDuplicates(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	db.Postgresql.Exec("DELETE FROM credit_usages WHERE 1 = 1")
	db.Postgresql.Exec("DELETE FROM org_roles WHERE 1 = 1")
	db.Postgresql.Exec("DELETE FROM invitations WHERE 1 = 1")
	db.Postgresql.Unscoped().Where("1 = 1").Delete(&models.OrganisationPlan{})
	db.Postgresql.Unscoped().Where("1 = 1").Delete(&models.Plan{})

	if err := db.Postgresql.AutoMigrate(&models.Plan{}, &models.Organisation{}, &models.OrganisationPlan{}); err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	seed.SeedPlans(logger, db.Postgresql)
	seed.SeedRolesAndPermissions(logger, db.Postgresql)

	var plan models.Plan
	err := db.Postgresql.Where("name = ?", "Free").First(&plan).Error
	assert.NoError(t, err)

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:            orgID,
		Name:          "test org",
		Email:         "test@example.com",
		Description:   "test description",
		Type:          "test",
		Location:      "test location",
		Country:       "test country",
		OwnerID:       utility.GenerateUUID(),
		CreditBalance: 100.0,
		OrgPlanID:     plan.ID,
	}

	err = db.Postgresql.Create(&org).Error
	assert.NoError(t, err)

	orgPlan1 := models.OrganisationPlan{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		PlanID:         plan.ID,
		StartedAt:      time.Now().Add(-60 * 24 * time.Hour),
		EndedAt:        time.Now().Add(-30 * 24 * time.Hour),
		Status:         "Inactive",
		SessionID:      "session_1",
	}
	err = db.Postgresql.Create(&orgPlan1).Error
	assert.NoError(t, err)

	orgPlan2 := models.OrganisationPlan{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		PlanID:         plan.ID,
		StartedAt:      time.Now().Add(-30 * 24 * time.Hour),
		EndedAt:        time.Time{},
		Status:         "Active",
		SessionID:      "session_2",
	}
	err = db.Postgresql.Create(&orgPlan2).Error
	assert.NoError(t, err)

	subscriptions, code, err := subscription.GetSubscriptions(orgID, db.Postgresql)

	assert.NoError(t, err)
	assert.Equal(t, 200, code)
	assert.NotNil(t, subscriptions)

	assert.Equal(t, 2, len(subscriptions), "Expected exactly 2 subscription records")

	sessionIDs := make(map[string]bool)
	for _, sub := range subscriptions {
		if sub.SessionID != "" {
			if sessionIDs[sub.SessionID] {
				t.Errorf("Duplicate session_id found: %s", sub.SessionID)
			}
			sessionIDs[sub.SessionID] = true
		}
	}

	assert.Equal(t, 2, len(sessionIDs), "Expected 2 unique session IDs, found duplicates")

	assert.Equal(t, "Active", subscriptions[0].Status, "First subscription should be Active (most recent)")
	assert.Equal(t, "Inactive", subscriptions[1].Status, "Second subscription should be Inactive (older)")
}
