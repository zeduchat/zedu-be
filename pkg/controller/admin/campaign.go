package admin

import (
	"github.com/gin-gonic/gin"
	contract "github.com/hngprojects/telex_be/internal/campaign"
	"github.com/hngprojects/telex_be/services/campaign"
	"net/http"
)

// Registered only inside the existing AdminAuthorize backoffice group.
func (base *Controller) CampaignReport(c *gin.Context) {
	start, end, err := contract.DateRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from/to must be YYYY-MM-DD, inclusive, at most 90 days"})
		return
	}
	if err := campaign.PurgeExpired(base.Db.Postgresql); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "report unavailable"})
		return
	}
	rows, err := campaign.Report(base.Db.Postgresql, start, end)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "report unavailable"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"data": rows, "model": "first-touch, 30-day attributed visit cohorts; unattributed/direct excluded"})
}
