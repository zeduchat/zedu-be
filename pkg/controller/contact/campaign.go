package contact

import (
	"github.com/gin-gonic/gin"
	contract "github.com/hngprojects/telex_be/internal/campaign"
	"github.com/hngprojects/telex_be/services/campaign"
	"net/http"
)

func (base *Controller) CampaignVisit(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
	var a contract.Attribution
	if c.ShouldBindJSON(&a) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign visit"})
		return
	}
	if err := campaign.Record(base.Db.Postgresql, &a, "visit", ""); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "measurement unavailable"})
		return
	}
	c.Status(http.StatusNoContent)
}
