package subscriptions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/subscription"
	"github.com/hngprojects/telex_be/utility"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/webhook"
)

func (base *Controller) HandleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Unable to read request body", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		base.Logger.Error(err.Error())
		return
	}

	var config = config.GetConfig()
	stripeWebhookSecret := config.Stripe.STRIPE_WEBHOOK_SECRET

	event, err := webhook.ConstructEvent(body, c.Request.Header.Get("Stripe-Signature"), stripeWebhookSecret)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid webhook signature", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		base.Logger.Error(err.Error())
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

		var orgEmail string
		if checkoutSession.CustomerDetails != nil {
			orgEmail = checkoutSession.CustomerDetails.Email
		} else {
			orgEmail = checkoutSession.Customer.Email
		}

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

		req := models.CompleteSubscriptionRequest{
			Email:           orgEmail,
			StripeSessionID: checkoutSession.ID,
		}

		_, code, _, err := service.CompleteSubscriptionWebhook(req, base.Db.Postgresql)
		if err != nil {
			rd := utility.BuildErrorResponse(code, "error", "Something went wrong", err.Error(), nil)
			c.JSON(code, rd)
			base.Logger.Error(fmt.Sprintf("Subscription completion failed for session %s: %v", checkoutSession.ID, err.Error()))
			return
		}

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

	base.Logger.Info("subscription created")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Subscription created successfully", nil)
	c.JSON(http.StatusOK, rd)
}
