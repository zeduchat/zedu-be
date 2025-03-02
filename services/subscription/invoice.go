package subscription

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func DownloadInvoice(sessionID string, c *gin.Context, db *gorm.DB, rdb *redis.Client, orgID string) error {

	req := models.CompleteSubscriptionRequest{
		OrgID:           orgID,
		StripeSessionID: sessionID,
	}
	_, _, inv, err := CompleteSubscription(req, db, rdb)
	if err != nil {
		return err
	}

	pdfURL := inv.InvoicePDF
	if pdfURL == "" {
		return errors.New("PDF not available for this invoice")
	}

	resp, err := http.Get(pdfURL)
	if err != nil {
		return errors.New("failed to download PDF")
	}
	defer resp.Body.Close()

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=invoice_%s.pdf", inv.Number))
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")
	c.Header("Cache-Control", "must-revalidate")
	c.Header("Pragma", "public")

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return errors.New("failed to stream PDF")
	}

	return nil
}
