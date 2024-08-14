package models

import (
	"gorm.io/gorm"
)

type SubscriptionPlan struct {
	Name          string  `gorm:"type:varchar(100);not null"`
	Price         float64 `gorm:"type:decimal(10,2);not null"`
	Description   string  `gorm:"type:text"`
	Features      string  `gorm:"type:text"`
	StripePriceID string  `gorm:"type:varchar(255);not null"`
}

func SeedSubscriptionPlans(db *gorm.DB) {
	plans := []SubscriptionPlan{
		{
			Name:          "Basic",
			Price:         20.00,
			Description:   "The essential tools to produce your best work for clients.",
			Features:      "Basic features",
			StripePriceID: "price_1Pnqb1JlOh7AbM5NLNQ0UTEs",
		},
		{
			Name:          "Advanced",
			Price:         50.00,
			Description:   "The essential tools to produce your best work for clients.",
			Features:      "Advanced features",
			StripePriceID: "price_1PnqbTJlOh7AbM5NMlbwHHdX",
		},
		{
			Name:          "Premium",
			Price:         100.00,
			Description:   "The essential tools to produce your best work for clients.",
			Features:      "Premium features",
			StripePriceID: "price_1PnqbFJlOh7AbM5NzYKR3lC7",
		},
	}

	for _, plan := range plans {
		db.FirstOrCreate(&plan, SubscriptionPlan{Name: plan.Name})
	}
}

type CreateSubscriptionRequest struct {
	PlanName      string            `json:"plan_name" binding:"required"`
	UserID        string            `json:"user_id" binding:"required"`
	Email         string            `json:"email" binding:"required,email"`
	PlanID        string            `json:"plan_id" binding:"required"`
	PaymentMethod string            `json:"payment_method" binding:"required"`
	Currency      string            `json:"currency" binding:"required"`
	BillingCycle  string            `json:"billing_cycle" binding:"required"`
	AddressLine1  string            `json:"address_line1,omitempty"`
	AddressLine2  string            `json:"address_line2,omitempty"`
	City          string            `json:"city,omitempty"`
	State         string            `json:"state,omitempty"`
	PostalCode    string            `json:"postal_code,omitempty"`
	Country       string            `json:"country,omitempty"`
	CouponCode    string            `json:"coupon_code,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ModifySubscriptionRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	PlanName string `json:"plan_name" binding:"required"`
	PlanID   uint   `json:"plan_id" binding:"required"`
}

type DeleteSubscriptionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}
