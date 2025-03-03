package subscription

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func DownloadInvoice(sessionID string, c *gin.Context, db *gorm.DB, rdb *redis.Client, orgID string) (gin.H, error) {

	req := models.CompleteSubscriptionRequest{
		OrgID:           orgID,
		StripeSessionID: sessionID,
	}
	_, _, inv, err := CompleteSubscription(req, db, rdb)
	if err != nil {
		return gin.H{}, err
	}

	pdfURL := inv.InvoicePDF
	if pdfURL == "" {
		return gin.H{}, errors.New("PDF not available for this invoice")
	}

	url := gin.H{
		"invoice_url": pdfURL,
	}

	return url, nil
}
