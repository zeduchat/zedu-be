package models

import (
	"errors"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type OrgUserManagement struct {
	UserID         string    `gorm:"type:uuid;primaryKey;not null" json:"user_id"`
	OrganisationID string    `gorm:"type:uuid;primaryKey;not null" json:"organisation_id"`
	Status         string    `gorm:"type:varchar(255)" json:"status"`
	RoleID         int64     `gorm:"type:int;not null" json:"role_id"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	DeletedAt      time.Time `gorm:"index" json:"deleted_at"`
}

type OrgUserManagementResponse struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	OrganisationID string `json:"organisation_id"`
	RoleID         string `json:"role_id"`
	Status         string `json:"status"`
	Role           Role   `json:"role"`
}

type OrgUserManagementRequest struct {
	UserID         string `json:"user_id" validate:"required"`
	OrganisationID string `json:"organisation_id" validate:"required"`
	RoleID         string `json:"role_id" validate:"required"`
	Status         string `json:"status" validate:"required"`
}

type OrgUserManagementUpdateRequest struct {
	Status string `json:"status" validate:"required"`
	RoleID string `json:"role_id" validate:"required"`
}

type OrgUserMetricsResponse struct {
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
	TotalMembers  int64 `json:"total_members"`
}

type UpdateMemberRequest struct {
	Status string `json:"status"`
	RoleID *int64 `json:"role_id"`
}

func (o *OrgUserManagement) CreateOrgUserManagement(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &o)
	if err != nil {
		return err
	}

	return nil
}

func (o *OrgUserManagement) GetOrgUserManagement(db *gorm.DB, users []UserInOrgResponse, orgID string) ([]UserInOrgResponse, error) {
	var orgUserManagement OrgUserManagement
	var response []UserInOrgResponse

	for _, user := range users {
		_, _ = postgresql.SelectOneFromDb(db, &orgUserManagement, "organisation_id = ? AND user_id = ?", orgID, user.ID)
		user.Role = orgUserManagement.RoleID
		user.Status = orgUserManagement.Status

		response = append(response, user)
	}

	return response, nil
}

func (o *OrgUserManagement) CountMetrics(db *gorm.DB, orgID string) (OrgUserMetricsResponse, error) {

	exists := postgresql.CheckExists(db, o, "organisation_id = ?", orgID)
	if !exists {
		return OrgUserMetricsResponse{}, errors.New("organisation not found")
	}

	activeCount, _ := postgresql.CountSpecificRecords(db, o, "organisation_id = ? AND status = ?", orgID, "active")
	inactiveCount, _ := postgresql.CountSpecificRecords(db, o, "organisation_id = ? AND status = ?", orgID, "inactive")
	totalMembers, _ := postgresql.CountSpecificRecords(db, o, "organisation_id = ?", orgID)

	countData := OrgUserMetricsResponse{
		ActiveCount:   activeCount,
		InactiveCount: inactiveCount,
		TotalMembers:  totalMembers,
	}

	return countData, nil
}

func (o *OrgUserManagement) UpdateMember(db *gorm.DB, orgID, userID string, req UpdateMemberRequest) (OrgUserManagement, error) {
	var oum OrgUserManagement

	exists := postgresql.CheckExists(db, &o, "organisation_id = ? AND user_id = ?", orgID, userID)
	if !exists {
		return oum, errors.New("user not found in organisation")
	}

	err, _ := postgresql.SelectOneFromDb(db, &o, "organisation_id = ? AND user_id = ?", orgID, userID)
	if err != nil {
		return oum, errors.New("user not found in organisation")
	}

	if req.Status != "" {
		o.Status = req.Status
	}

	if req.RoleID != nil {
		o.RoleID = *req.RoleID
	}

	result, err := postgresql.SaveAllFields(db, &o)
	if err != nil {
		return oum, errors.New("failed to update user")
	}

	if result.RowsAffected == 0 {
		return oum, errors.New("failed to update user")
	}

	oum = OrgUserManagement{
		UserID:         o.UserID,
		OrganisationID: o.OrganisationID,
		Status:         o.Status,
		RoleID:         o.RoleID,
	}

	return oum, nil
}

func (o *OrgUserManagement) RemoveMemberFromOrganisation(db *gorm.DB, orgID, userID string) error {

	var (
		u  User
		og Organisation
	)

	exists := postgresql.CheckExists(db, o, "organisation_id = ? AND user_id = ?", orgID, userID)
	if !exists {
		return errors.New("user not found in organisation")
	}

	err, _ := postgresql.SelectOneFromDb(db, &o, "organisation_id = ? AND user_id = ?", orgID, userID)
	if err != nil {
		return errors.New("user not found in organisation")
	}

	err = postgresql.DeleteRecordFromDb(db, &o)
	if err != nil {
		return errors.New("failed to remove user from organisation")
	}

	err, _ = postgresql.SelectOneFromDb(db, &u, "id = ?", userID)
	if err != nil {
		return errors.New("user not found")
	}

	err, _ = postgresql.SelectOneFromDb(db, &og, "id = ?", orgID)
	if err != nil {
		return errors.New("organisation not found")
	}

	err = u.RemoveUserFromOrganisation(db, &u, []interface{}{&og})
	if err != nil {
		return errors.New("failed to remove user from organisation")
	}

	return nil
}
