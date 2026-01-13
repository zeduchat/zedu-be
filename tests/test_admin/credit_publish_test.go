package test_admin

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestPublishPlatformCreditUpdate_Success(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	CreateCreditTransaction(t, db.Postgresql, orgID, 100.00)
	CreateCreditUsage(t, db.Postgresql, orgID, 20.00)

	logger := utility.NewLogger()

	models.PublishPlatformCreditUpdate(db.Postgresql, logger)

	t.Cleanup(func() {
		CleanupTestData(t, db.Postgresql)
	})

	metrics, err := models.GetPlatformCreditSummary(db.Postgresql)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, metrics.TotalCredited, float64(100.00))
	assert.GreaterOrEqual(t, metrics.TotalUsed, float64(20.00))
	assert.GreaterOrEqual(t, metrics.TotalBalance, float64(80.00))
}
