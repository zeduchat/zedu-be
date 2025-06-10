package credits

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"

	// service "github.com/hngprojects/telex_be/services/credits"
	"github.com/hngprojects/telex_be/utility"
	"github.com/stripe/stripe-go/v72"
)

func (base *Controller) HandleStripeCreditPaymentWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		base.Logger.Error(err.Error())
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Unable to read request body", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	var event stripe.Event
	if err := json.Unmarshal(body, &event); err != nil {
		base.Logger.Error("Error parsing event: " + err.Error())
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid event data", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		var checkoutSession stripe.CheckoutSession

		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			base.Logger.Error("Error parsing webhook JSON:" + err.Error())
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Error parsing webhook JSON", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		if !strings.HasPrefix(checkoutSession.SuccessURL, "https://telex.im") &&
			!strings.HasPrefix(checkoutSession.SuccessURL, "https://staging.telex.im") {
			base.Logger.Info("Webhook not for telex")
			rd := utility.BuildSuccessResponse(http.StatusOK, "Webhook not for telex", nil)
			c.JSON(http.StatusOK, rd)
			return
		}

		// check if webhook has been processed
		var webhookData = models.ProcessedStripeWebhook{}
		isProcessed, err := webhookData.IsWebhookProcessed(base.Db.Postgresql, checkoutSession.ID)
		if err != nil {
			base.Logger.Error(fmt.Sprintf("Error checking webhook processing status: %v", err.Error()))
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Error checking webhook processing status", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		if isProcessed {
			base.Logger.Info(fmt.Sprintf("Session ID %s has already been processed", checkoutSession.ID))
			rd := utility.BuildSuccessResponse(http.StatusOK, "Webhook already processed", nil)
			c.JSON(http.StatusOK, rd)
			return
		}

		// refill credit here
		_ = checkoutSession.Metadata["org_id"]
		_ = checkoutSession.Metadata["flow"]

		// Mark webhhook as processed
		newWebhookData := models.ProcessedStripeWebhook{
			ID:          utility.GenerateUUID(),
			SessionID:   checkoutSession.ID,
			ProcessedAt: time.Now(),
		}

		err = newWebhookData.MarkWebhookAsProcessed(base.Db.Postgresql)
		if err != nil {
			base.Logger.Error(fmt.Sprintf("Error marking webhook as processed: %v", err.Error()))
			rd := utility.BuildSuccessResponse(http.StatusOK, "success", nil)
			c.JSON(http.StatusOK, rd)
			return
		}

	default:
		base.Logger.Info(fmt.Sprintf("Unhandled event type: %v", event.Type))
		rd := utility.BuildSuccessResponse(http.StatusOK, "Event not handled", nil)
		c.JSON(http.StatusOK, rd)
		return
	}

	base.Logger.Info("Credit Purchased")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Credit Purchased successfully", nil)
	c.JSON(http.StatusOK, rd)
}
