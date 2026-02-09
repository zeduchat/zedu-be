package subscriptions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	service "github.com/hngprojects/telex_be/services/subscription"
	"github.com/hngprojects/telex_be/utility"
	"github.com/stripe/stripe-go/v72"
)

func (base *Controller) HandleStripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		base.Logger.Error(err.Error())
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Unable to read request body", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	base.Logger.Info("DEBUG: Webhook body: %s\n", string(body))
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

		rawObject := event.Data.Raw
		if len(rawObject) == 0 && event.Data.Object != nil {
			rawObject, _ = json.Marshal(event.Data.Object)
		}

		err := json.Unmarshal(rawObject, &checkoutSession)
		if err != nil {
			base.Logger.Error("Error parsing webhook JSON:" + err.Error())
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Error parsing webhook JSON", nil, nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}
		base.Logger.Info("DEBUG: Webhook session ID: %s, SuccessURL: %s\n", checkoutSession.ID, checkoutSession.SuccessURL)

		if !strings.HasPrefix(checkoutSession.SuccessURL, "https://zedu.chat") &&
			!strings.HasPrefix(checkoutSession.SuccessURL, "https://staging.zedu.chat") &&
			!strings.HasPrefix(checkoutSession.SuccessURL, "https://staging.zedu.im") {
			base.Logger.Info("Webhook not for telex")
			rd := utility.BuildSuccessResponse(http.StatusOK, "Webhook not for telex", nil)
			c.JSON(http.StatusOK, rd)
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

		switch checkoutSession.Metadata["flow"] {
		case "credit_topup":
			org_id := checkoutSession.Metadata["org_id"]
			plan_id := checkoutSession.Metadata["plan_id"]

			_, code, err := models.TopUpOrgCredit(base.Db.Postgresql, org_id, plan_id, &checkoutSession.ID)
			if err != nil {
				rd := utility.BuildErrorResponse(code, "error", "Something went wrong", err.Error(), nil)
				c.JSON(code, rd)
				base.Logger.Error(fmt.Sprintf("Credit top-up failed for session %s: %v", checkoutSession.ID, err.Error()))
				return
			}

		case "subscription":
			plan_id := checkoutSession.Metadata["plan_id"]
			org_id := checkoutSession.Metadata["org_id"]
			stripe_customer_id := checkoutSession.Metadata["customer_id"]

			req := models.CompleteSubscriptionRequest{
				Email:            orgEmail,
				StripeSessionID:  checkoutSession.ID,
				PlanID:           plan_id,
				OrgID:            org_id,
				StripeCustomerID: stripe_customer_id,
			}

			_, code, _, err := service.CompleteSubscriptionWebhook(req, base.Db.Postgresql)
			if err != nil {
				rd := utility.BuildErrorResponse(code, "error", "Something went wrong", err.Error(), nil)
				c.JSON(code, rd)
				base.Logger.Error(fmt.Sprintf("Subscription completion failed for session %s: %v", checkoutSession.ID, err.Error()))
				return
			}
		default:
			base.Logger.Info("Unhandled flow: %s", checkoutSession.Metadata["flow"])
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
