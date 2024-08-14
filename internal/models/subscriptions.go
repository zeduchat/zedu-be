package models

import "gorm.io/gorm"

type SubscriptionPlan struct {
	gorm.Model
	Name        string  `gorm:"type:varchar(100);not null"`
	Price       float64 `gorm:"type:decimal(10,2);not null"`
	Description string  `gorm:"type:text"`
	Features    string  `gorm:"type:text"`
}

func SeedSubscriptionPlans(db *gorm.DB) {
	plans := []SubscriptionPlan{
		{
			Name:        "Basic",
			Price:       20.00,
			Description: "The essential tools to produce your best work for clients.",
			Features:    "Basic features",
		},
		{
			Name:        "Advanced",
			Price:       50.00,
			Description: "The essential tools to produce your best work for clients.",
			Features:    "Advanced features",
		},
		{
			Name:        "Premium",
			Price:       100.00,
			Description: "The essential tools to produce your best work for clients.",
			Features:    "Premium features",
		},
	}

	for _, plan := range plans {
		db.FirstOrCreate(&plan, SubscriptionPlan{Name: plan.Name})
	}
}

type CreateSubscriptionRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	PlanName string `json:"plan_name" binding:"required"`
	PlanID   uint   `json:"plan_id" `
}

type ModifySubscriptionRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	PlanName string `json:"plan_name" binding:"required"`
	PlanID   uint   `json:"plan_id" binding:"required"`
}

type DeleteSubscriptionRequest struct {
	UserID string `json:"user_id" binding:"required"`
}
