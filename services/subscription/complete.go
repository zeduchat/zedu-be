package subscription

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/invoice"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

func CompleteSubscription(req models.CompleteSubscriptionRequest, db *gorm.DB,
	rdb *redis.Client) (*gin.H, int, *stripe.Invoice, error) {
	var orgRepo models.Organisation
	var planRepo models.Plan
	var orgPlanRepo models.OrganisationPlan

	org, err := orgRepo.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, nil, errors.New("org not found")
	}

	sesh, err := session.Get(req.StripeSessionID, nil)
	if err != nil {
		return nil, http.StatusBadRequest, nil, errors.New("error getting session")
	}

	theAmt := int(sesh.AmountSubtotal) / 100
	if sesh.PaymentStatus != "paid" {
		return nil, http.StatusBadRequest, nil, errors.New("session not paid")
	}

	if sesh.Subscription == nil || sesh.Subscription.ID == "" {
		return nil, http.StatusBadRequest, nil, errors.New("no subscription ID found")
	}

	plan, err := planRepo.GetAPlanByAmount(db, theAmt)
	if err != nil {
		return nil, http.StatusBadRequest, nil, errors.New("plan not found")
	}

	orgPlan, err := orgPlanRepo.GetAnOrgPlanById(db, req.OrgID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, http.StatusInternalServerError, nil, errors.New("error retrieving organisation plan")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {

		orgPlan = models.OrganisationPlan{
			ID:             utility.GenerateUUID(),
			OrganisationID: req.OrgID,
			PlanID:         plan.ID,
			StartedAt:      time.Now(),
			Status:         "Active",
			SessionID:      req.StripeSessionID,
		}
		org.OrgPlanID = orgPlan.ID
		org.OrganisationPlan = orgPlan

		err = orgPlan.Create(db)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("error creating organisation plan")
		}
	} else {

		orgPlan.EndedAt = time.Now()
		orgPlan.Status = "Inactive"
		_, err = orgPlan.Update(db)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("error updating existing organisation plan")
		}

		newOrgPlan := models.OrganisationPlan{
			ID:             utility.GenerateUUID(),
			OrganisationID: req.OrgID,
			PlanID:         plan.ID,
			StartedAt:      time.Now(),
			Status:         "Active",
			SessionID:      req.StripeSessionID,
		}

		org.OrgPlanID = newOrgPlan.ID
		org.OrganisationPlan = newOrgPlan

		err = newOrgPlan.Create(db)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("error creating new organisation plan")
		}
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

	cacheKey := "org_plan:" + org.ID
	rd.RedisDelete(rdb, cacheKey)

	responseData := gin.H{
		"invoice_items": invoiceItems,
	}

	return &responseData, http.StatusOK, invoiceItems[0], nil
}
