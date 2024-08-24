package subscription

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/product"
	"github.com/stripe/stripe-go/v72/sub"
	"gorm.io/gorm"
)

func CreateSubscription(req *models.CreateSubscriptionRequest, db *gorm.DB, url string) (*gin.H, int, error) {

	var subscriptionPlan models.SubscriptionPlan
	if err := db.Where("name = ?", req.PlanName).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, errors.New("subscription plan not found")
	}

	if subscriptionPlan.Name == "" {
		return nil, http.StatusInternalServerError, errors.New("subscription plan name is missing")
	}

	if subscriptionPlan.StripePriceID == "" {
		return nil, http.StatusInternalServerError, errors.New("missing StripePriceID for subscription plan")
	}

	log.Printf("Subscription Plan: %v", subscriptionPlan)

	stripeCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}
	stripeCustomer, err := customer.New(stripeCustomerParams)
	if err != nil {
		log.Printf("Error creating Stripe customer: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to create Stripe customer")
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomer.ID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(subscriptionPlan.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(url + "/dashboard/settings/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "/dashboard/plan/billing"),
	}

	session, err := session.New(params)
	if err != nil {
		log.Printf("Error creating Stripe checkout session: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to create checkout session")
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}
	return &responseData, http.StatusOK, nil
}

func ListSubscriptions(customerID string, db *gorm.DB) (*gin.H, int, error) {
	var user models.User
	if err := db.First(&user, "id = ?", customerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, http.StatusNotFound, errors.New("user not found")
		}
		return nil, http.StatusInternalServerError, errors.New("internal server error")
	}

	subscription, err := sub.Get(user.SubscriptionPlanId, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to retrieve subscription")
	}

	var subscriptionInfo gin.H
	if len(subscription.Items.Data) > 0 {
		item := subscription.Items.Data[0]
		productID := item.Price.Product.ID
		price := item.Price.UnitAmountDecimal
		product, err := product.Get(productID, nil)
		if err != nil {
			return nil, http.StatusInternalServerError, errors.New("failed to retrieve product")
		}

		startDate := time.Unix(subscription.StartDate, 0).Format("January 2, 2006")
		var endDate string
		if subscription.CancelAt > 0 {
			endDate = time.Unix(subscription.CancelAt, 0).Format("January 2, 2006")
		} else if subscription.CurrentPeriodEnd > 0 {
			endDate = time.Unix(subscription.CurrentPeriodEnd, 0).Format("January 2, 2006")
		} else {
			endDate = "Not set"
		}

		subscriptionInfo = gin.H{
			"name":            product.Name,
			"start_date":      startDate,
			"end_date":        endDate,
			"subscription_id": subscription.ID,
			"price":           price,
		}
	} else {
		return nil, http.StatusNotFound, errors.New("no subscription items found")
	}

	responseData := gin.H{
		"subscription": subscriptionInfo,
	}
	return &responseData, http.StatusOK, nil
}

func ModifySubscription(req *models.ModifySubscriptionRequest, db *gorm.DB, url string) (*gin.H, int, error) {
	var subscriptionPlan models.SubscriptionPlan
	if err := db.Where("name = ?", req.PlanName).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, errors.New("subscription plan not found")
	}

	if subscriptionPlan.StripePriceID == "" {
		return nil, http.StatusInternalServerError, errors.New("missing StripePriceID for subscription plan")
	}

	var user models.User
	if err := db.Where("id = ?", req.UserID).First(&user).Error; err != nil {
		return nil, http.StatusNotFound, errors.New("user not found")
	}

	if user.StripeCustomerID == "" {
		return nil, http.StatusInternalServerError, errors.New("user does not have a Stripe customer ID")
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(user.StripeCustomerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(subscriptionPlan.StripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(url + "/dashboard/plan/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "/dashboard/plan/billing/"),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to create checkout session")
	}

	user.SubscriptionPlanId = subscriptionPlan.StripePriceID
	if err := db.Save(&user).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("error updating user subscription")
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}

	return &responseData, http.StatusOK, nil
}

func DeleteSubscription(userId string, db *gorm.DB) (int, error) {
	var user models.User
	if err := db.First(&user, "id = ?", userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusNotFound, errors.New("user not found")
		}
		return http.StatusInternalServerError, errors.New("internal server error")
	}

	if user.SubscriptionPlanId == "" {
		return http.StatusBadRequest, errors.New("user has no subscription plan")
	}

	_, err := sub.Cancel(user.SubscriptionPlanId, nil)
	if err != nil {
		log.Printf("Error cancelling subscription: %v", err)
		return http.StatusInternalServerError, errors.New("error cancelling subscription")
	}

	user.SubscriptionPlanId = ""

	if err := db.Save(&user).Error; err != nil {
		return http.StatusInternalServerError, errors.New("error updating user subscription plan")
	}

	return http.StatusOK, nil
}

func CompleteSubscription(session_id, user_id string, db *gorm.DB) (*gin.H, int, error, *stripe.Invoice) {
	var user models.User
	if err := db.First(&user, "id = ?", user_id).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("error finding user"), nil
	}

	sesh, err := session.Get(session_id, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("error getting session"), nil
	}

	if sesh.PaymentStatus != "paid" {
		return nil, http.StatusBadRequest, errors.New("session not paid"), nil
	}

	if sesh.Subscription == nil || sesh.Subscription.ID == "" {
		return nil, http.StatusBadRequest, errors.New("no subscription ID found"), nil
	}

	user.SubscriptionPlanId = sesh.Subscription.ID
	user.StripeCustomerID = sesh.Customer.ID

	if err := db.Save(&user).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("error updating user subscription"), nil
	}

	params := &stripe.InvoiceListParams{
		Subscription: stripe.String(sesh.Subscription.ID),
	}

	i := invoice.List(params)

	var invoiceItems []*stripe.Invoice

	for i.Next() {
		invoiceItems = append(invoiceItems, i.Invoice())
	}

	if err := i.Err(); err != nil {
		return nil, http.StatusInternalServerError, errors.New("error retrieving invoice"), nil
	}

	responseData := gin.H{
		"invoice_items": invoiceItems,
	}

	return &responseData, http.StatusOK, nil, invoiceItems[0]
}
