package subscription

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func DownloadInvoice(sessionID string, c *gin.Context, db *gorm.DB, user string) error {
	_, _, err, inv := CompleteSubscription(sessionID, user, db)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
