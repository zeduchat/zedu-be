package subscription

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/invoice"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func DownloadInvoice(sessionID string, c *gin.Context, db *gorm.DB, rdb *redis.Client, orgID string) (gin.H, error) {

	req := models.CompleteSubscriptionRequest{
		StripeSessionID: sessionID,
	}

	sesh, err := session.Get(req.StripeSessionID, nil)
	if err != nil {
		return gin.H{}, err
	}

	params := &stripe.InvoiceListParams{
		Subscription: stripe.String(sesh.Subscription.ID),
	}

	i := invoice.List(params)

	var invoiceItems []*stripe.Invoice

	for i.Next() {
		invoiceItems = append(invoiceItems, i.Invoice())
	}

	pdfURL := invoiceItems[0].InvoicePDF
	if pdfURL == "" {
		return gin.H{}, errors.New("PDF not available for this invoice")
	}

	url := gin.H{
		"invoice_url": pdfURL,
	}

	return url, nil
}
