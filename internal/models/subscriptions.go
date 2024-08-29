package models

import (
	"github.com/hngprojects/telex_be/internal/config"
)

var StripeMap map[string]string

func SetStripeMap(stripeConfig config.Stripe) {
	StripeMap = map[string]string{
		"Basic":    stripeConfig.STRIPE_BASIC_ID,
		"Advanced": stripeConfig.STRIPE_ADVANCED_ID,
		"Premium":  stripeConfig.STRIPE_PREMIUM_ID,
	}
}

type CreateSubscriptionRequest struct {
	PlanName string `json:"plan_name" validate:"required"`
	OrgID    string `json:"org_id" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
}

type ModifySubscriptionRequest struct {
	OrgID    string `json:"org_id" validate:"required"`
	PlanName string `json:"plan_name" validate:"required"`
}

type DeleteSubscriptionRequest struct {
	OrgID string `json:"org_id" validate:"required"`
}

type CompleteSubscriptionRequest struct {
	OrgID           string `json:"org_id" validate:"required"`
	StripeSessionID string `json:"stripe_session_id" validate:"required"`
}
