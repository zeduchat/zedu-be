package subscription

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func DownloadInvoice(sessionID string, c *gin.Context) error {
	var db *storage.Database
	_, _, err, inv := CompleteSubscription(sessionID, "", db.Postgresql)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	pdfURL := inv.InvoicePDF
	if pdfURL == "" {
		return fmt.Errorf("PDF not available for this invoice")
	}

	resp, err := http.Get(pdfURL)
	if err != nil {
		return fmt.Errorf("failed to download PDF: %v", err)
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
		return fmt.Errorf("failed to stream PDF: %v", err)
	}

	return nil
}
