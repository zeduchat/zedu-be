package models

import (
	"github.com/hngprojects/telex_be/internal/config"
	"gorm.io/gorm"
)

type SubscriptionPlan struct {
	Name          string  `gorm:"type:varchar(100);not null"`
	Price         float64 `gorm:"type:decimal(10,2);not null"`
	Description   string  `gorm:"type:text"`
	Features      string  `gorm:"type:text"`
	StripePriceID string  `gorm:"type:varchar(255);not null"`
}

func SeedSubscriptionPlans(db *gorm.DB, stripe config.Stripe) {
	plans := []SubscriptionPlan{
		{
			Name:          "Basic",
			Price:         20.00,
			Description:   "The essential tools to produce your best work for clients.",
			Features:      "Basic features",
			StripePriceID: stripe.STRIPE_BASIC_ID,
		},
		{
			Name:          "Advanced",
			Price:         50.00,
			Description:   "The essential tools to produce your best work for clients.",
			Features:      "Advanced features",
			StripePriceID: stripe.STRIPE_ADVANCED_ID,
		},
		{
			Name:          "Premium",
			Price:         100.00,
			Description:   "The essential tools to produce your best work for clients.",
			Features:      "Premium features",
			StripePriceID: stripe.STRIPE_PREMIUM_ID,
		},
	}

	for _, plan := range plans {
		db.FirstOrCreate(&plan, SubscriptionPlan{Name: plan.Name})
	}
}

type CreateSubscriptionRequest struct {
	PlanName string `json:"plan_name" binding:"required"`
	UserID   string `json:"user_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type ModifySubscriptionRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	PlanName string `json:"plan_name" binding:"required"`
	PlanID   uint   `json:"plan_id" binding:"required"`
}

type DeleteSubscriptionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}
