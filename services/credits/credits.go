package credits

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

	stripeCustomerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}

	stripeCustomer, err := customer.New(stripeCustomerParams)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	priceInCents := int64(credit.Price * 100)

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(stripeCustomer.ID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(strings.ToLower(credit.Currency)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(credit.Name),
						Description: stripe.String("Credit package with " + strconv.Itoa(credit.Credits) + " credits"),
					},
					UnitAmount: stripe.Int64(priceInCents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(url + "client/settings/organisation/billing?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(url + "client/settings/organisation/billing"),
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
func VerifyPayment(sessionID string, db *gorm.DB) (*gin.H, int, error) {

	var existingTransaction models.CreditTransaction
	err := db.Where("stripe_session_id = ?", sessionID).First(&existingTransaction).Error
	if err == nil {
		return nil, http.StatusOK, nil
	}

	stripeSession, err := session.Get(sessionID, nil)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to retrieve stripe session: %v", err)
	}

	if stripeSession.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		return nil, http.StatusBadRequest, errors.New("payment not completed")
	}

	orgID := stripeSession.Metadata["org_id"]
	packageID := stripeSession.Metadata["package_id"]

	if orgID == "" || packageID == "" {
		return nil, http.StatusBadRequest, errors.New("invalid session metadata")
	}

	return models.TopUpOrgCredit(db, orgID, packageID, &sessionID)
}
