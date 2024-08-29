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
	PlanName string `json:"plan_name" binding:"required" validate:"required"`
	OrgID    string `json:"org_id" binding:"required" validate:"required"`
	Email    string `json:"email" binding:"required,email" validate:"required"`
}

type ModifySubscriptionRequest struct {
	OrgID    string `json:"org_id" binding:"required" validate:"required"`
	PlanName string `json:"plan_name" binding:"required" validate:"required"`
}

type DeleteSubscriptionRequest struct {
	OrgID string `json:"org_id" binding:"required" validate:"required"`
}

type CompleteSubscriptionRequest struct {
	OrgID           string `json:"org_id" binding:"required" validate:"required"`
	StripeSessionID string `json:"stripe_session_id" binding:"required" validate:"required"`
}
