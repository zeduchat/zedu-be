package credits

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/checkout/session"
	"github.com/stripe/stripe-go/v72/customer"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

func PurchaseCredits(req models.CreditTopUpRequest, db *gorm.DB, url string) (*gin.H, int, error) {
	var org models.Organisation

	org, err := org.GetOrgByID(db, req.OrgID)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	credit, _, err := models.GetCreditPackageByID(db, req.PackageID)
	if err != nil {
		return nil, http.StatusNotFound, err
	}

	stripePriceID, exists := GetStripePriceID(credit.Name)
	if !exists {
		return nil, http.StatusBadRequest, errors.New("missing StripePriceID for credit plan")
	}

	stripeCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}

	stripeCustomer, err := customer.New(stripeCustomerParams)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomer.ID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(stripePriceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(url + "clients?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "clients"),
	}

	params.AddMetadata("org_id", req.OrgID)
	params.AddMetadata("flow", "credit_topup")
	params.AddMetadata("package_id", req.PackageID)

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

func GetStripePriceID(planName string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(planName))
	priceID, exist := models.MapPackagePriceID[normalized]

	if !exist {
		return priceID, false
	}

	return priceID, true
}
