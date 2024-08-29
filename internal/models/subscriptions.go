package models

import (
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"gorm.io/gorm"
)

var StripeMap map[string]string

func SetStripeMap(stripeConfig config.Stripe) {
	StripeMap = map[string]string{
		"Starter":    stripeConfig.STRIPE_BASIC_ID,
		"Business":   stripeConfig.STRIPE_ADVANCED_ID,
		"Enterprise": stripeConfig.STRIPE_PREMIUM_ID,
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

type Plan struct {
	ID                      string         `gorm:"primaryKey;type:uuid" json:"id"`
	Name                    string         `gorm:"uniqueIndex;" json:"name"`
	Fee                     int            `gorm:"not null" json:"fee"`
	MaxChannels             int            `gorm:"not null" json:"max_channels"`
	MaxUsers                int            `gorm:"not null" json:"max_users"`
	MaxNotifications        int            `gorm:"not null" json:"max_notifications"`
	CanUpgradeNotifications bool           `gorm:"not null" json:"can_upgrade_notifications"`
	CanAddUnlimitedChannels bool           `gorm:"not null" json:"can_add_unlimited_channels"`
	CanAddUnlimitedUsers    bool           `gorm:"not null" json:"can_add_unlimited_users"`
	IsForIndividuals        bool           `gorm:"not null" json:"is_for_individuals"`
	IsForSmallBusiness      bool           `gorm:"not null" json:"is_for_small_business"`
	IsForLargeEnterprise    bool           `gorm:"not null" json:"is_for_large_enterprise"`
	CreatedAt               time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`
}

type OrganisationPlan struct {
	ID             string         `gorm:"primaryKey;type:uuid" json:"id"`
	OrganisationID string         `gorm:"not null;index" json:"organisation_id"`
	PlanID         string         `gorm:"not null;index" json:"plan_id"`
	StartedAt      time.Time      `gorm:"column:started_at;null; autoCreateTime" json:"started_at"`
	EndedAt        *time.Time     `gorm:"column:ended_at; null" json:"ended_at"`
	Status         string         `gorm:"null" json:"status"`
	CreatedAt      time.Time      `gorm:"column:created_at; null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (r *OrganisationPlan) GetAPlanById(db *gorm.DB, orgID string) (OrganisationPlan, error) {
	var orgPlan OrganisationPlan

	query := db.Where("organisation_id = ? AND status = ?", orgID, "Active")
	err := query.First(&orgPlan).Error

	if err != nil {
		return orgPlan, err
	}

	return orgPlan, nil
}
