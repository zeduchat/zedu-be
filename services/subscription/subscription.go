package subscription

import (
	"errors"
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

func CreateSubscription(req models.CreateSubscriptionRequest, db *gorm.DB,
	url string) (*gin.H, int, error) {
	var org models.Organisation

	org, err := org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	stripePriceID, exists := models.StripeMap[req.PlanName]
	if !exists {
		return nil, http.StatusBadRequest, errors.New("missing StripePriceID for subscription plan")
	}

	stripeCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}

	stripeCustomer, err := customer.New(stripeCustomerParams)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to create Stripe customer")
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomer.ID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(stripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(url + "dashboard/settings/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "dashboard/settings/billing"),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}

	return &responseData, http.StatusOK, nil
}

func ListSubscriptions(customerID string, db *gorm.DB) (*gin.H, int, error) {
	var org models.Organisation
	org, err := org.GetOrgByID(db, customerID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	subscription, err := sub.Get(org.SubscriptionPlanId, nil)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to retrieve subscription")
	}

	var subscriptionInfo gin.H
	if len(subscription.Items.Data) > 0 {
		item := subscription.Items.Data[0]
		productID := item.Price.Product.ID
		price := item.Price.UnitAmountDecimal
		product, err := product.Get(productID, nil)
		if err != nil {
			return nil, http.StatusBadRequest, errors.New("failed to retrieve product")
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

func ModifySubscription(req models.ModifySubscriptionRequest, db *gorm.DB, url string) (*gin.H, int, error) {

	stripePriceID, exists := models.StripeMap[req.PlanName]
	if !exists {
		return nil, http.StatusBadRequest, errors.New("missing StripePriceID for subscription plan")
	}

	var org models.Organisation
	org, err := org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	if org.StripeCustomerID == "" {
		return nil, http.StatusBadRequest, errors.New("org does not have a Stripe customer ID")
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(org.StripeCustomerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(stripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(url + "dashboard/settings/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "dashboard/settings/billing/"),
	}

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to create checkout session")
	}

	org.SubscriptionPlanId = session.Subscription.ID
	org.StripeCustomerID = session.Customer.ID

	_, err = org.Update(db)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("error updating org subscription")
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}

	return &responseData, http.StatusOK, nil
}

func DeleteSubscription(orgId string, db *gorm.DB) (int, error) {
	var org models.Organisation
	org, err := org.GetOrgByID(db, orgId)
	if err != nil {
		return http.StatusNotFound, errors.New("org not found")
	}

	if org.SubscriptionPlanId == "" {
		return http.StatusBadRequest, errors.New("org has no subscription plan")
	}

	_, err = sub.Cancel(org.SubscriptionPlanId, nil)
	if err != nil {
		return http.StatusBadRequest, errors.New("error cancelling subscription")
	}

	org.SubscriptionPlanId = ""

	_, err = org.Update(db)
	if err != nil {
		return http.StatusBadRequest, errors.New("error updating org subscription plan")
	}

	return http.StatusOK, nil
}

func CompleteSubscription(req models.CompleteSubscriptionRequest, db *gorm.DB) (*gin.H, int, *stripe.Invoice, error) {
	var org models.Organisation
	org, err := org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, nil, errors.New("org not found")
	}

	sesh, err := session.Get(req.StripeSessionID, nil)
	if err != nil {
		return nil, http.StatusBadRequest, nil, errors.New("error getting session")
	}

	if sesh.PaymentStatus != "paid" {
		return nil, http.StatusBadRequest, nil, errors.New("session not paid")
	}

	if sesh.Subscription == nil || sesh.Subscription.ID == "" {
		return nil, http.StatusBadRequest, nil, errors.New("no subscription ID found")
	}

	org.SubscriptionPlanId = sesh.Subscription.ID
	org.StripeCustomerID = sesh.Customer.ID

	_, err = org.Update(db)
	if err != nil {
		return nil, http.StatusBadRequest, nil, errors.New("error updating org subscription plan")
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
		return nil, http.StatusBadRequest, nil, errors.New("error retrieving invoice")
	}

	responseData := gin.H{
		"invoice_items": invoiceItems,
	}

	return &responseData, http.StatusOK, invoiceItems[0], nil
}
