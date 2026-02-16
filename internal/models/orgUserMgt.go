package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type OrgUserManagement struct {
	UserID         string                 `gorm:"type:uuid;primaryKey;not null" json:"user_id"`
	OrganisationID string                 `gorm:"type:uuid;primaryKey;not null" json:"organisation_id"`
	Status         string                 `gorm:"type:varchar(255)" json:"status"`
	RoleID         string                 `gorm:"type:uuid;null" json:"role_id"`
	IsDeactivated  bool                   `gorm:"column:is_deactivated;type:bool;default:false" json:"is_deactivated"`
	CreatedAt      time.Time              `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	DeletedAt      time.Time              `gorm:"index" json:"deleted_at"`
	Preferences    NotificationPreference `gorm:"type:jsonb;not null;default:'{}'" json:"preferences"`
}

type OrgUserCreateRequest struct {
	RoleID string `json:"role_id" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
}

type OrgGetStartedResponse struct {
	UserName       string                         `json:"user_name"`
	OrgUserProfile []OrgUsersProfile              `json:"org_users_profile"`
	OrgChannel     []OrgChannelsWithMemberAvatars `json:"org_channels"`
}

type OrgUsersProfile struct {
	Name      string `json:"name"`
	AvatarUrl string `json:"avatar_url"`
	IsOnline  bool   `json:"is_online"`
}

type OrgChannelsWithMemberAvatars struct {
	Name          string   `json:"name"`
	MemberAvatars []string `json:"member_avatars"`
	LastPostTime  string   `json:"last_post_time"` // Formatted as "a few seconds ago", "2 minutes ago", etc.
	MembersCount  int      `json:"members_count"`
	UnreadCount   int      `json:"unread_count,omitempty"`
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
	UserID         string    `json:"user_id" validate:"required"`
	OrganisationID string    `json:"organisation_id" validate:"required"`
	RoleID         string    `json:"role_id" validate:"required"`
	Status         string    `json:"status" validate:"required"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	DeletedAt      time.Time `gorm:"index" json:"deleted_at"`
}

type OrgUserManagementUpdateRequest struct {
	Status string `json:"status" validate:"required"`
	RoleID string `json:"role_id" validate:"required"`
}

type OrgUserMetricsResponse struct {
	ActiveCount   int64 `json:"active_count"`
	InactiveCount int64 `json:"inactive_count"`
	TotalMembers  int64 `json:"total_members"`
	TotalGuests   int64 `json:"total_guests"`
}

type UpdateMemberRequest struct {
	Status string `json:"status"`
	RoleID string `json:"role_id"`
}

type OrgUserRoleInfo struct {
	RoleID         string   `json:"role_id"`
	RoleName       string   `json:"role_name"`
	Permissions    []string `json:"permissions,omitempty"`
	OrganisationID string   `json:"-"`
}

type ChangeMemberActiveStatus struct {
	Activate bool `gorm:"activate" json:"activate"`
}

type UpdateMemberRoleRequest struct {
	RoleID string `json:"role_id" validate:"required"`
}

func (o *OrgUserManagement) CreateOrgUserManagement(db *gorm.DB) error {

	err := postgresql.CreateOneRecord(db, &o)
	if err != nil {
		return err
	}
	return nil
}

func (o *OrgUserManagement) GetOrgUserManagement(db *gorm.DB, users []UserInOrgResponse, orgID string) ([]UserInOrgResponse, error) {
	var response []UserInOrgResponse
	var orgUserManagement OrgUserManagement
	var userstable User
	var org_role OrgRole

	for _, user := range users {
		_, _ = postgresql.SelectOneFromDb(db, &orgUserManagement, "organisation_id = ? AND user_id = ?", orgID, user.ID)
		_, _ = postgresql.SelectOneFromDb(db, &userstable, "id = ?", user.ID)
		_, _ = postgresql.SelectOneFromDb(db, &org_role, "id = ?", userstable.OrgRoleID)
		user.Role = org_role.Name
		user.Status = orgUserManagement.Status

		response = append(response, user)
	}
	return response, nil
}

func (o *OrgUserManagement) CountMetrics(db *gorm.DB, orgID string) (OrgUserMetricsResponse, error) {
	var (
		inv Invitation
	)

	exists := postgresql.CheckExists(db, o, "organisation_id = ?", orgID)
	if !exists {
		return OrgUserMetricsResponse{}, errors.New("organisation not found")
	}

	activeCount, _ := postgresql.CountSpecificRecords(db, o, "organisation_id = ? AND status = ?", orgID, "active")
	inactiveCount, _ := postgresql.CountSpecificRecords(db, o, "organisation_id = ? AND status = ?", orgID, "inactive")
	totalMembers, _ := postgresql.CountSpecificRecords(db, o, "organisation_id = ?", orgID)
	totalGuests, _ := postgresql.CountSpecificRecords(db, inv, "organisation_id = ? AND status = ?", orgID, "accepted")

	countData := OrgUserMetricsResponse{
		ActiveCount:   activeCount,
		InactiveCount: inactiveCount,
		TotalMembers:  totalMembers,
		TotalGuests:   totalGuests,
	}
	return countData, nil
}

func (o *OrgUserManagement) GetByIDs(db *gorm.DB, userID, orgID string) (OrgUserManagement, error) {

	query := db.Where("user_id = ? AND organisation_id = ?", userID, orgID)
	if err := query.First(&o).Error; err != nil {
		return *o, err
	}

	return *o, nil
}

func (o *OrgUserManagement) Update(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &o)
	return err
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

	if req.RoleID != "" {
		o.RoleID = req.RoleID
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

	err = u.RemoveUserFromOrganisation(db, &u, []any{&og})
	if err != nil {
		return errors.New("failed to remove user from organisation")
	}

	return nil
}

func (o *OrgUserManagement) AddUserToOrganisation(db *gorm.DB) (int, error) {
	var (
		user User
		org  Organisation
	)

	user, err := user.GetUserByID(db, o.UserID)
	if err != nil {
		return 404, fmt.Errorf("failed to get user: %w", err)
	}

	org, err = org.GetOrgByID(db, o.OrganisationID)
	if err != nil {
		return 404, fmt.Errorf("failed to get organisation: %w", err)
	}

	exists := postgresql.CheckExists(db, &o, "organisation_id = ? AND user_id = ?", o.OrganisationID, o.UserID)
	if exists {
		return 409, fmt.Errorf("user already exists in organisation")
	}

	err = postgresql.CreateOneRecord(db, &o)
	if err != nil {
		return 500, fmt.Errorf("failed to create org user management: %w", err)
	}

	err = user.AddUserToOrganisation(db, &user, []any{&org})
	if err != nil {
		return 500, fmt.Errorf("failed to add user to organisation: %w", err)
	}

	return 201, nil
}

func (o *OrgUserManagement) UpdateAllOrgUsersWithNewRole(db *gorm.DB, orgID, roleID string) error {
	var role OrgRole
	defaultRole, err := role.GetAOrgRoleByName(db, "User")
	if err != nil {
		return err
	}

	orgUserManagementUpdate := postgresql.ModelUpdate{
		Model:   &OrgUserManagement{},
		Updates: map[string]any{"role_id": defaultRole.ID},
		Where:   "organisation_id = ? AND role_id = ?",
		Args:    []any{orgID, roleID},
	}

	defaultRoleID, err := uuid.FromString(defaultRole.ID)
	if err != nil {
		return errors.New("invalid role id")
	}

	userUpdate := postgresql.ModelUpdate{
		Model:   &User{},
		Updates: map[string]any{"org_role_id": defaultRoleID},
		Where:   "users.id IN (SELECT o.user_id FROM org_user_managements AS o WHERE o.organisation_id = ? AND o.role_id = ?)",
		Args:    []any{orgID, roleID},
	}

	return postgresql.UpdateFieldsInTransaction(db, []postgresql.ModelUpdate{orgUserManagementUpdate, userUpdate})
}

func (o *OrgUserManagement) GetUserRoleInOrganisation(db *gorm.DB, userID, orgID string) (OrgUserRoleInfo, error) {
	var userRoleInfo OrgUserRoleInfo

	var result struct {
		RoleID         string
		OrganisationID string
		RoleName       string
		PermissionList PermissionList
	}

	err := db.Table("org_user_managements").
		Select(`
        org_user_managements.role_id,
        org_user_managements.organisation_id,
        org_roles.name AS role_name,
		permissions.permission_list
    `).
		Joins("LEFT JOIN org_roles ON org_user_managements.role_id = org_roles.id").
		Joins("LEFT JOIN permissions ON org_roles.id = permissions.role_id").
		Where("org_user_managements.user_id = ?", userID).
		Where("org_user_managements.organisation_id = ?", orgID).
		Scan(&result).Error

	if err != nil {
		return userRoleInfo, fmt.Errorf("failed to get user role in organisation: %v", err)
	}

	userRoleInfo = OrgUserRoleInfo{
		RoleID:         result.RoleID,
		RoleName:       result.RoleName,
		OrganisationID: result.OrganisationID,
		Permissions:    result.PermissionList.ToSlice(),
	}

	return userRoleInfo, nil
}

func (o *OrgUserManagement) CheckIsUserDeactivated(db *gorm.DB, ids IDS) bool {
	var orgUserManagement OrgUserManagement

	return postgresql.CheckExists(db, &orgUserManagement, "organisation_id = ? AND user_id = ? AND is_deactivated = ?", ids.OrganisationID, ids.UserID, true)
}

func (o *OrgUserManagement) SearchUsersInOrganisation(db *gorm.DB, orgID, searchTerm string) ([]UserInOrgResponse, error) {
	var users []UserInOrgResponse

	query := db.Table("users").
		Select(`
            users.id,
            users.email,
            profiles.user_name AS username,
            profiles.phone AS phone_number,
            profiles.avatar_url AS profile_url,
            users.name,
            org_roles.name AS role,
            org_user_managements.status,
            users.created_at,
            'user' AS entity_type,
            profiles.online
        `).
		Joins("JOIN org_user_managements ON org_user_managements.user_id = users.id").
		Joins("LEFT JOIN org_roles ON org_user_managements.role_id = org_roles.id").
		Joins("LEFT JOIN profiles ON profiles.userid = users.id").
		Where("org_user_managements.organisation_id = ?", orgID).
		Where(`
            users.name ILIKE ? OR 
            users.email ILIKE ? OR 
            profiles.user_name ILIKE ? OR 
            profiles.phone ILIKE ?
        `, "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%")

	if err := query.Scan(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to search users in organisation: %v", err)
	}

	return users, nil
}

func (oum *OrgUserManagement) UpdateMemberRole(db *gorm.DB, ids IDS) error {
	var oumCheck OrgUserManagement

	exists := postgresql.CheckExists(db, &oumCheck, "organisation_id = ? AND user_id = ?", ids.OrganisationID, ids.UserID)
	if !exists {
		return errors.New("user not found in organisation")
	}

	update := map[string]any{
		"role_id": ids.RoleID,
	}

	res, err := postgresql.UpdateFields(db, &oumCheck, update, "organisation_id = ? AND user_id = ?", ids.OrganisationID, ids.UserID)
	if err != nil {
		return fmt.Errorf("unable to update user role: %w", err)
	}

	if res.RowsAffected == 0 {
		return errors.New("no record updated")
	}

	return nil
}

func (oum *OrgUserManagement) CheckIsOrganisationAdmin(db *gorm.DB, ids IDS) bool {
	var orgUserManagement OrgUserManagement

	query := "organisation_id = ? AND user_id = ? AND role_id = (SELECT id FROM org_roles WHERE name = 'Administrator')"

	return postgresql.CheckExists(db, &orgUserManagement, query, ids.OrganisationID, ids.OwnerID)
}
