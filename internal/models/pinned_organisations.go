package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type UserPinnedOrganisations struct {
	ID        string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	UserID    string         `gorm:"type:uuid;not null" json:"user_id"`
	OrgID     string         `gorm:"type:uuid;not null" json:"org_id"`
	CreatedAt time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateUserPinnedOrganisationRequest struct {
	OrgID string `json:"org_id" validate:"required"`
}

type GetUserPinnedOrganisationsResponse struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	OrgName   string    `json:"org_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UnpinOrganisationRequest struct {
	OrgID string `json:"org_id" validate:"required"`
}

func (u *UserPinnedOrganisations) CheckOrganisationPinned(db *gorm.DB, ids IDS) bool {
	var pinnedOrg UserPinnedOrganisations

	return postgresql.CheckExists(db, &pinnedOrg, "user_id = ? AND org_id = ?", ids.UserID, ids.OrganisationID)
}

func (u *UserPinnedOrganisations) CreateUserPinnedOrganisation(db *gorm.DB) error {
	return postgresql.CreateOneRecord(db, &u)
}

func (u *UserPinnedOrganisations) GetUserPinnedOrganisations(db *gorm.DB, ids IDS) ([]GetUserPinnedOrganisationsResponse, error) {
	var pinnedOrgs []GetUserPinnedOrganisationsResponse

	err := db.Table("user_pinned_organisations").
		Select("user_pinned_organisations.id, user_pinned_organisations.org_id, organisations.name as org_name, user_pinned_organisations.created_at, user_pinned_organisations.updated_at").
		Joins("JOIN organisations ON organisations.id = user_pinned_organisations.org_id").
		Where("user_pinned_organisations.user_id = ?", ids.UserID).
		Order("user_pinned_organisations.created_at DESC").
		Scan(&pinnedOrgs).Error

	if err != nil {
		return nil, errors.New("failed to get user pinned organisations: " + err.Error())
	}

	return pinnedOrgs, nil
}

func (u *UserPinnedOrganisations) UnpinOrganisation(db *gorm.DB, ids IDS) error {
	return postgresql.HardDeleteSpecificRecord(db, &u, "user_id = ? AND org_id = ?", ids.UserID, ids.OrganisationID)
}

func (u *UserPinnedOrganisations) RemoveOldestPinnedOrganisation(db *gorm.DB, userID string) error {
	// If we have pinned organizations, get the oldest one
	err := db.Where("user_id = ?", userID).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("no pinned organisations found")
		}
		return fmt.Errorf("failed to get oldest pinned organisation: %w", err)
	}

	// return postgresql.HardDeleteSpecificRecord(db, &u, "user_id = ? AND org_id = ?", u.UserID, u.OrgID)
	return nil
}

func (u *UserPinnedOrganisations) CountCurrentUserPinnedOrganisations(db *gorm.DB, ids IDS) (int, error) {
	count, err := postgresql.CountSpecificRecords(db, &u, "user_id = ?", ids.UserID)
	if err != nil {
		return 0, errors.New("failed to count current user pinned organisations: " + err.Error())
	}

	return int(count), nil
}
