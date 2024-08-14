package subscription

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/sub"
	"gorm.io/gorm"
)

func CreateSubscription(req *models.CreateSubscriptionRequest, db *gorm.DB) (*gin.H, int, error) {
	var subscriptionPlan models.SubscriptionPlan
	if err := db.Where("name = ?", req.PlanName).First(&subscriptionPlan).Error; err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("subscription plan not found: %v", err)
	}

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
		SuccessURL: stripe.String("https://staging.telex.im/dashboard/plan/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String("https://yourwebsite.com/plan/billing/cancel"),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create checkout session: %v", err)
	}

	if err := db.Model(&models.User{}).Where("id = ?", req.UserID).Updates(models.User{
		StripeCustomerID: stripeCustomer.ID,
	}).Error; err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update user with Stripe customer ID: %v", err)
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}

	return &responseData, http.StatusOK, nil
}

func ListSubscriptions(customerID string, db *gorm.DB) (*gin.H, int, error) {
	params := &stripe.SubscriptionListParams{
		Customer: customerID,
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

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(req.UserID),
				Price: stripe.String(subscriptionPlan.Name),
			},
		},
	}

	subscription, err := sub.Update(req.UserID, params)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	responseData := gin.H{
		"subscription_id": subscription.ID,
		"status":          subscription.Status,
		"plan":            subscriptionPlan.Name,
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
	_, err := sub.Cancel(strconv.FormatUint(uint64(user.SubscriptionPlan.ID), 10), nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error cancelling subscription: %w", err)
	}
	if err := db.Model(&user).Association("SubscriptionPlan").Clear(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error removing subscription plan: %w", err)
	}
	return http.StatusOK, nil
}
