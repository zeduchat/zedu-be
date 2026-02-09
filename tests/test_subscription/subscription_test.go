package test_subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/subscription"
	tst "github.com/hngprojects/telex_be/tests"
)

func TestGetSubscriptionPlans_Fields(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	db.Postgresql.Unscoped().Where("1 = 1").Delete(&models.Plan{})

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
