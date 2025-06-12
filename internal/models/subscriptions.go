package models

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
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
	Email           string `json:"email"`
	OrgID           string `json:"org_id"`
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
	Credits                 int            `gorm:"not null;default:0" json:"credits"`
	UpdatedAt               time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`
}

type OrganisationPlan struct {
	ID             string         `gorm:"primaryKey;type:uuid" json:"id"`
	OrganisationID string         `gorm:"not null;index" json:"organisation_id"`
	PlanID         string         `gorm:"not null;index" json:"plan_id"`
	StartedAt      time.Time      `gorm:"column:started_at;null; autoCreateTime" json:"started_at"`
	EndedAt        time.Time      `gorm:"column:ended_at; null" json:"ended_at"`
	Status         string         `gorm:"null" json:"status"`
	SessionID      string         `gorm:"null" json:"session_id"`
	CreatedAt      time.Time      `gorm:"column:created_at; null; autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProcessedStripeWebhook struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	SessionID   string    `gorm:"uniqueIndex;"`
	ProcessedAt time.Time `gorm:"null"`
}

func (c *OrganisationPlan) Create(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &c)
	if err != nil {
		return err
	}
	return nil
}

func (c *OrganisationPlan) Update(db *gorm.DB) (*OrganisationPlan, error) {
	result, err := postgresql.SaveAllFields(db, &c)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected == 0 {
		return nil, errors.New("failed to update organisation plan")
	}

	return c, nil
}

func (r *OrganisationPlan) GetAnOrgPlanById(db *gorm.DB, orgID string) (OrganisationPlan, error) {
	var orgPlan OrganisationPlan

	query := db.Where("organisation_id = ? AND status = ?", orgID, "Active").
		Order("started_at DESC")
	err := query.First(&orgPlan).Error

	if err != nil {
		return orgPlan, err
	}

	return orgPlan, nil
}

func (r *OrganisationPlan) GetPlanByOrgID(db *gorm.DB, orgID string) (Plan, error) {
	var plan Plan

	err := db.Table("organisation_plans").
		Select("plans.*").
		Joins("JOIN plans ON organisation_plans.plan_id::uuid = plans.id").
		Where("organisation_plans.organisation_id = ? AND organisation_plans.status = ?", orgID, "Active").
		Order("organisation_plans.started_at DESC").
		First(&plan).Error

	if err != nil {
		return plan, err
	}

	return plan, nil
}

func (r *Plan) GetAPlanByAmount(db *gorm.DB, planAmt int) (Plan, error) {

	query := db.Where("fee = ?", planAmt)
	err := query.First(&r).Error

	if err != nil {
		return *r, err
	}

	return *r, nil
}

type OrgPlanDetails struct {
	ID                    string    `json:"id,omitempty"`
	Name                  string    `json:"name"`
	Fee                   float64   `json:"fee"`
	StartDate             time.Time `json:"start_date"`
	EndDate               time.Time `json:"end_date"`
	Status                string    `json:"status"`
	SessionID             string    `json:"session_id"`
	OrganisationCreatedAt time.Time `json:"organisation_created_at"`
}

func (r *OrganisationPlan) GetOrgPlanDetailByOrgID(db *gorm.DB, orgID string) (OrgPlanDetails, error) {
	var details OrgPlanDetails

	query := `
        SELECT p.name AS name, 
               p.fee AS fee, 
               op.started_at AS start_date, 
               op.ended_at AS end_date,
               o.created_at AS organisation_created_at,
			   op.status AS  status
        FROM organisations o
        LEFT JOIN organisation_plans op ON o.id = op.organisation_id AND op.status = ?
        LEFT JOIN plans p ON op.plan_id::uuid = p.id
        WHERE o.id = ?
        ORDER BY op.started_at DESC
        LIMIT 1;
    `
	err := db.Raw(query, "Active", orgID).Scan(&details).Error
	if err != nil {
		return details, err
	}

	if details.StartDate.IsZero() && details.EndDate.IsZero() {

		StartDate := details.OrganisationCreatedAt

		err := db.Raw(query, "Inactive", orgID).Scan(&details).Error
		if err != nil {
			return details, err
		}

		if !(details.StartDate.IsZero() && details.EndDate.IsZero()) {
			StartDate = details.EndDate

		}

		details = OrgPlanDetails{
			Name:                  "Free",
			Fee:                   0.0,
			Status:                "Active",
			StartDate:             StartDate,
			EndDate:               StartDate.AddDate(0, 0, 30),
			OrganisationCreatedAt: details.OrganisationCreatedAt,
		}
	}

	return details, nil
}

func (r *OrganisationPlan) GetOrgPlanDetailsByOrgID(db *gorm.DB, orgID string) ([]OrgPlanDetails, error) {
	var details []OrgPlanDetails

	query := `
        SELECT p.name AS name, 
               p.fee AS fee, 
               op.started_at AS start_date, 
               op.ended_at AS end_date,
               o.created_at AS organisation_created_at,
			   op.status AS  status,
			   op.session_id AS session_id,
			   op.id AS id
        FROM organisations o
        LEFT JOIN organisation_plans op ON o.id = op.organisation_id
        LEFT JOIN plans p ON op.plan_id::uuid = p.id
        WHERE o.id = ?
        ORDER BY op.started_at DESC;
    `
	err := db.Raw(query, orgID).Scan(&details).Error
	if err != nil {
		return details, err
	}

	if len(details) == 1 && details[0].StartDate.IsZero() && details[0].EndDate.IsZero() {
		details = []OrgPlanDetails{}
	}

	return details, nil
}

func (r *ProcessedStripeWebhook) IsWebhookProcessed(db *gorm.DB, sessionID string) (bool, error) {

	err := db.Where("session_id = ?", sessionID).First(&r).Error
	if err == nil {
		return true, nil
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else {
		return false, err
	}
}

func (r *ProcessedStripeWebhook) MarkWebhookAsProcessed(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &r)
	if err != nil {
		return err
	}
	return nil
}

func GetSubscriptionPlans(db *gorm.DB) (*[]Plan, error) {
	var subPlans []Plan

	if err := db.Find(&subPlans).Error; err != nil {
		return nil, err
	}

	return &subPlans, nil
}
