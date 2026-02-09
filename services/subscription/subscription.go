package subscription

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/product"
	"github.com/stripe/stripe-go/v72/sub"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func CreateSubscription(req models.CreateSubscriptionRequest, db *gorm.DB,
	url string) (*gin.H, int, error) {

	plan, err := models.GetPlanByID(db, req.PlanID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("plan not found")
	}

	stripeCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}

	stripeCustomer, err := customer.New(stripeCustomerParams)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to create Stripe customer")
	}

	priceInCents := int64(plan.Fee * 100)
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomer.ID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(plan.Name),
						Description: stripe.String(plan.Description),
					},
					UnitAmount: stripe.Int64(priceInCents),
					Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval: stripe.String("month"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(url + "client/settings/organisation/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "client/settings/organisation/billing"),
	}

	params.AddMetadata("flow", "subscription")
	params.AddMetadata("plan_id", req.PlanID)
	params.AddMetadata("plan_name", plan.Name)
	params.AddMetadata("org_id", req.OrgID)
	params.AddMetadata("customer_id", stripeCustomer.ID)

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

	plan, err := models.GetPlanByID(db, req.PlanID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("plan not found")
	}

	var org models.Organisation
	org, err = org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("org not found")
	}

	if org.StripeCustomerID == "" {
		return nil, http.StatusBadRequest, errors.New("org does not have a Stripe customer ID")
	}

	priceInCents := int64(plan.Fee * 100)
	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(org.StripeCustomerID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(plan.Name),
						Description: stripe.String(plan.Description),
					},
					UnitAmount: stripe.Int64(priceInCents),
					Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
						Interval: stripe.String("month"),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(url + "client/settings/organisation/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "client/settings/organisation/billing/"),
	}

	params.AddMetadata("flow", "subscription")
	params.AddMetadata("plan_id", req.PlanID)
	params.AddMetadata("plan_name", plan.Name)
	params.AddMetadata("org_id", req.OrgID)
	params.AddMetadata("customer_id", org.StripeCustomerID)

	session, err := session.New(params)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New("failed to create checkout session")
	}

	responseData := gin.H{
		"checkout_session_id":  session.ID,
		"checkout_session_url": session.URL,
	}

	return &responseData, http.StatusOK, nil
}

func DeleteSubscription(orgId string, db *gorm.DB) (int, error) {
	var org models.Organisation
	var orgPlanRepo models.OrganisationPlan

	org, err := org.GetOrgByID(db, orgId)
	if err != nil {
		return http.StatusNotFound, errors.New("org not found")
	}

	if org.OrgPlanID == "" {
		return http.StatusBadRequest, errors.New("org has no subscription plan")
	}

	orgPlan, err := orgPlanRepo.GetAnOrgPlanById(db, org.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return http.StatusBadRequest, errors.New("error retrieving organisation plan")
	}

	org.OrgPlanID = ""
	org.OrganisationPlan = models.OrganisationPlan{}

	_, err = org.Update(db)
	if err != nil {
		return http.StatusBadRequest, errors.New("error updating org subscription plan")
	}

	orgPlan.EndedAt = time.Now()
	orgPlan.Status = "Inactive"
	_, err = orgPlan.Update(db)
	if err != nil {
		return http.StatusBadRequest, errors.New("error updating existing organisation plan")
	}

	return http.StatusOK, nil
}

func GetCurrentSubscription(customerID string, db *gorm.DB) (models.OrgPlanDetails, int, error) {
	var org models.Organisation
	var orgPlan models.OrganisationPlan
	var currentPlan models.OrgPlanDetails

	org, err := org.GetOrgByID(db, customerID)
	if err != nil {
		return currentPlan, http.StatusNotFound, errors.New("org not found")
	}

	currentPlan, err = orgPlan.GetOrgPlanDetailByOrgID(db, customerID)
	if err != nil {
		return currentPlan, http.StatusNotFound, errors.New("failed to retrieve")
	}

	currentPlan.EndDate = currentPlan.StartDate.AddDate(0, 0, 30)

	return currentPlan, http.StatusOK, nil
}

func GetSubscriptions(customerID string, db *gorm.DB) ([]models.OrgPlanDetails, int, error) {
	var org models.Organisation
	var orgPlan models.OrganisationPlan
	var currentPlans []models.OrgPlanDetails

	org, err := org.GetOrgByID(db, customerID)
	if err != nil {
		return currentPlans, http.StatusNotFound, errors.New("org not found")
	}

	currentPlans, err = orgPlan.GetOrgPlanDetailsByOrgID(db, customerID)
	if err != nil {
		return currentPlans, http.StatusNotFound, errors.New("failed to retrieve")
	}

	return currentPlans, http.StatusOK, nil
}

func GetSubscriptionPlans(db *gorm.DB) (*[]models.PlanResponse, int, error) {

	subscriptionPlans, err := models.GetSubscriptionPlans(db)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("failed to retrieve subscription plans")
	}

	return subscriptionPlans, http.StatusOK, nil
}
