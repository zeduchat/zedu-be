package subscription

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"gorm.io/gorm"
)

func CompleteSubscriptionWebhook(req models.CompleteSubscriptionRequest, db *gorm.DB) (*gin.H, int, *stripe.Invoice, error) {
	var orgRepo models.Organisation
	var planRepo models.Plan
	var orgPlanRepo models.OrganisationPlan

	org, err := orgRepo.GetOrgByEmail(db, req.Email)
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

	plan, err := planRepo.GetAPlanByAmount(db, theAmt)
	if err != nil {
		return nil, http.StatusBadRequest, nil, errors.New("plan not found")
	}

	orgPlan, err := orgPlanRepo.GetAnOrgPlanById(db, org.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, http.StatusInternalServerError, nil, errors.New("error retrieving organisation plan")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {

		orgPlan = models.OrganisationPlan{
			ID:             utility.GenerateUUID(),
			OrganisationID: org.ID,
			PlanID:         plan.ID,
			StartedAt:      time.Now(),
			EndedAt:        time.Now().AddDate(0, 0, 30),
			Status:         "Active",
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
			OrganisationID: org.ID,
			PlanID:         plan.ID,
			StartedAt:      time.Now(),
			EndedAt:        time.Now().AddDate(0, 0, 30),
			Status:         "Active",
		}

		org.OrgPlanID = newOrgPlan.ID
		org.OrganisationPlan = newOrgPlan

		err = newOrgPlan.Create(db)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("error creating new organisation plan")
		}
	}

	_, err = org.Update(db)
	if err != nil {
		return nil, http.StatusBadRequest, nil, errors.New("error updating org subscription plan")
	}

	return nil, http.StatusOK, nil, nil
}
