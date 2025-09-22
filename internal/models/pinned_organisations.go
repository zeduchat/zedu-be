package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type UserPinnedOrganisations struct {
	ID        string         `gorm:"column:id;type:uuid;primaryKey;unique;not null" json:"id"`
	UserID    string         `gorm:"column:user_id;type:uuid;not null" json:"user_id"`
	OrgID     string         `gorm:"column:org_id;type:uuid;not null" json:"org_id"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;null;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

type CreateUserPinnedOrganisationRequest struct {
	OrgID string `json:"org_id" validate:"required"`
}

type GetUserPinnedOrganisationsResponse struct {
	ID               string    `json:"id"`
	OrgID            string    `json:"org_id"`
	OrgName          string    `json:"org_name"`
	AvatarUrl        string    `json:"avatar_url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	OrganisationSlug string    `json:"organisation_slug"`
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
		Select("user_pinned_organisations.id, user_pinned_organisations.org_id, organisations.name as org_name, organisations.logo_url as avatar_url, user_pinned_organisations.created_at, user_pinned_organisations.updated_at").
		Joins("JOIN organisations ON organisations.id = user_pinned_organisations.org_id").
		Where("user_pinned_organisations.user_id = ?", ids.UserID).
		Order("user_pinned_organisations.created_at DESC").
		Scan(&pinnedOrgs).Error

	if err != nil {
		return nil, errors.New("failed to get user pinned organisations: " + err.Error())
	}

	for i := range pinnedOrgs {
		pinnedOrgs[i].OrganisationSlug = slug.Make(pinnedOrgs[i].OrgName)
	}

	return pinnedOrgs, nil
}

func (u *UserPinnedOrganisations) UnpinOrganisation(db *gorm.DB, ids IDS) error {
	return postgresql.HardDeleteSpecificRecord(db, &u, "user_id = ? AND org_id = ?", ids.UserID, ids.OrganisationID)
}

func (u *UserPinnedOrganisations) RemoveOldestPinnedOrganisation(db *gorm.DB, userID string) error {
	var oldestPinnedOrg UserPinnedOrganisations

	err := db.Table("user_pinned_organisations").
		Where("user_id = ?", userID).
		Order("created_at ASC").
		First(&oldestPinnedOrg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("no pinned organisations found")
		}
		return fmt.Errorf("failed to get oldest pinned organisation: %w", err)
	}

	return postgresql.HardDeleteSpecificRecord(db, &oldestPinnedOrg, "user_id = ? AND org_id = ?", oldestPinnedOrg.UserID, oldestPinnedOrg.OrgID)
}

func (u *UserPinnedOrganisations) CountCurrentUserPinnedOrganisations(db *gorm.DB, ids IDS) (int, error) {
	count, err := postgresql.CountSpecificRecords(db, &u, "user_id = ?", ids.UserID)
	if err != nil {
		return 0, errors.New("failed to count current user pinned organisations: " + err.Error())
	}

	return int(count), nil
}
