package stripe

import (
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func SetUpProducts(db *storage.Database, config config.Stripe) {
	models.SeedSubscriptionPlans(db.Postgresql, config)
}
