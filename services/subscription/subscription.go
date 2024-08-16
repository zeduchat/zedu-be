package subscription

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/invoice"
	"github.com/stripe/stripe-go/v72/sub"
	"gorm.io/gorm"
)

func CreateSubscription(req *models.CreateSubscriptionRequest, db *gorm.DB) (*gin.H, int, error) {
	var subscriptionPlan models.SubscriptionPlan
	if err := db.Where("name = ?", req.PlanName).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("subscription plan not found: %v", err)
	}

	if subscriptionPlan.StripePriceID == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("missing StripePriceID for subscription plan: %s", req.PlanName)
	}

	log.Print(subscriptionPlan)
	stripeCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}
	stripeCustomer, err := customer.New(stripeCustomerParams)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create Stripe customer: %v", err)
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
		SuccessURL: stripe.String("https://staging.telex.im/dashboard/settings/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("https://staging.telex.im/dashboard/plan/billing"),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create checkout session: %v", err)
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
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
	}

	params := &stripe.SubscriptionListParams{
		Customer: user.StripeCustomerID,
	}
	i := sub.List(params)

	var subscriptions []*stripe.Subscription
	for i.Next() {
		subscriptions = append(subscriptions, i.Subscription())
	}

	if err := i.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	responseData := gin.H{
		"subscriptions": subscriptions,
	}

	return &responseData, http.StatusOK, nil
}

func ModifySubscription(req *models.ModifySubscriptionRequest, db *gorm.DB) (*gin.H, int, error) {
	var subscriptionPlan models.SubscriptionPlan
	if err := db.Where("id = ?", req.PlanID).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("subscription plan not found: %v", err)
	}

	if subscriptionPlan.StripePriceID == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("missing StripePriceID for subscription plan: %v", req.PlanID)
	}

	var user models.User
	if err := db.Where("id = ?", req.UserID).First(&user).Error; err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("user not found: %v", err)
	}

	if user.StripeCustomerID == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("user does not have a Stripe customer ID")
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
		SuccessURL: stripe.String("https://staging.telex.im/dashboard/plan/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("https://yourwebsite.com/plan/billing/cancel"),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create checkout session: %v", err)
	}

	user.SubscriptionPlanId = user.SubscriptionPlanId
	if err := db.Save(&user).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error updating user subscription: %v", err)
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}

	return &responseData, http.StatusOK, nil
}

func DeleteSubscription(userId string, db *gorm.DB) (int, error) {
	var user *models.User
	if err := db.First(&user, "id = ?", userId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return http.StatusNotFound, fmt.Errorf("user not found")
		}
		return http.StatusInternalServerError, fmt.Errorf("error finding user: %w", err)
	}

	if user.SubscriptionPlanId == "" {
		return http.StatusBadRequest, fmt.Errorf("user has no subscription plan")
	}

	_, err := sub.Cancel(user.SubscriptionPlanId, nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error cancelling subscription: %w", err)
	}
	if err := db.Model(&user).Association("SubscriptionPlan").Clear(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error removing subscription plan: %w", err)
	}
	return http.StatusOK, nil
}

func CompleteSubscription(session_id, user_id string, db *gorm.DB) (*gin.H, int, error, *stripe.Invoice) {
	var user models.User
	if err := db.First(&user, "id = ?", user_id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, http.StatusNotFound, fmt.Errorf("user not found"), nil
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("error finding user: %w", err), nil
	}

	sesh, err := session.Get(session_id, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error getting session: %w", err), nil
	}

	if sesh.PaymentStatus != "paid" {
		return nil, http.StatusBadRequest, fmt.Errorf("session not paid"), nil
	}

	if sesh.Subscription == nil || sesh.Subscription.ID == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("no subscription ID found"), nil
	}

	user.SubscriptionPlanId = sesh.Subscription.ID
	user.StripeCustomerID = sesh.Customer.ID

	if err := db.Save(&user).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error updating user subscription: %w", err), nil
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
		return nil, http.StatusInternalServerError, err, nil
	}

	responseData := gin.H{
		"invoice_items": invoiceItems,
	}

	return &responseData, http.StatusOK, nil, invoiceItems[0]
}
