package test_admin

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func TestPublishPlatformCreditUpdate_Success(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	orgID := CreateOrganizationWithCredit(t, db.Postgresql, 100.00)
	CreateCreditTransaction(t, db.Postgresql, orgID, 100.00)
	CreateCreditUsage(t, db.Postgresql, orgID, 20.00)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, "", []string{orgID})
	})

	models.PublishPlatformCreditUpdate(db.Postgresql)

	metrics, err := models.GetPlatformCreditSummary(db.Postgresql)
	assert.NoError(t, err)
	assert.InDelta(t, metrics.TotalCredited-metrics.TotalUsed, metrics.TotalBalance, 0.01)
}
