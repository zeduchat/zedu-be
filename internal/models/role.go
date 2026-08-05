package models

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type RoleName string
type RoleId int

type DefaultIdentity struct {
	User       RoleId
	SuperAdmin RoleId
}

var RoleIdentity = DefaultIdentity{
	User:       1,
	SuperAdmin: 2,
}

type UserRole struct {
	Guest RoleId
	User  RoleId
	Admin RoleId
}

var (
	UserRoleName  RoleName = "user"
	AdminRoleName RoleName = "admin"
)

type Role struct {
	ID          int            `gorm:"primaryKey;type:int" json:"id"`
	Name        string         `gorm:"unique;not null;type:varchar(50)" json:"name" validate:"required"`
	Description string         `gorm:"unique;not null" json:"description" validate:"required"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type OrgRole struct {
	ID             string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name           string         `gorm:"null;type:varchar(20);unique" json:"name" validate:"required"`
	Description    string         `gorm:"null" json:"description" validate:"required"`
	OrganisationID *string        `gorm:"type:uuid;null" json:"organisation_id"`
	IsDefault      bool           `gorm:"type:bool" json:"is_default"`
	Permissions    Permission     `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE;" json:"permissions"`
	CreatedAt      time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"-"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"-"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateOrgRoleRequest struct {
	Name           string         `json:"name" validate:"required"`
	Description    string         `json:"description" validate:"required"`
	OrganisationID *string        `json:"organisation_id"`
	PermissionList PermissionList `json:"permission_list"`
	Permissions    PermissionList `json:"permissions"`
}

func (req *CreateOrgRoleRequest) GetPermissionList() PermissionList {
	if req.PermissionList != (PermissionList{}) {
		return req.PermissionList
	}
	return req.Permissions
}

type UpdateOrgPermissionsRequest struct {
	PermissionList                      PermissionList `json:"permission_list"`
	Permissions                         PermissionList `json:"permissions"`
	CanManageChannels                   bool           `json:"can_manage_channels"`
	CanManageMembers                    bool           `json:"can_manage_members"`
	CanManageOrganization               bool           `json:"can_manage_organization"`
	CanManageSettings                   bool           `json:"can_manage_settings"`
	CanManageBilling                    bool           `json:"can_manage_billing"`
	CanManageAgents                     bool           `json:"can_manage_agents"`
	CanManageWorkflows                  bool           `json:"can_manage_workflows"`
	CanManageIntegrations               bool           `json:"can_manage_integrations"`
	CanManageSecurity                   bool           `json:"can_manage_security"`
	CanManageRoles                      bool           `json:"can_manage_roles"`
	CanViewAnalytics                     bool           `json:"can_view_analytics"`
	CanViewBilling                      bool           `json:"can_view_billing"`
	CanViewChannels                     bool           `json:"can_view_channels"`
	CanEditMessages                     bool           `json:"can_edit_messages"`
	CanDeleteMessages                   bool           `json:"can_delete_messages"`
	CanDeleteFiles                      bool           `json:"can_delete_files"`
	CanCreateChannels                   bool           `json:"can_create_channels"`
	CanCreateAgents                     bool           `json:"can_create_agents"`
	CanCreateRole                       bool           `json:"can_create_role"`
	CanCreateWebhooks                   bool           `json:"can_create_webhooks"`
	CanArchiveChannels                  bool           `json:"can_archive_channels"`
	CanInviteMembers                    bool           `json:"can_invite_members"`
	CanRemovePeople                     bool           `json:"can_remove_people"`
	CanCommentThreads                   bool           `json:"can_comment_threads"`
	CanChangeUserOrgRole                bool           `json:"can_change_user_org_role"`
	CanCreateCustomRole                 bool           `json:"can_create_custom_role"`
	CanCreateChannel                    bool           `json:"can_create_channel"`
	CanCommentOnThreads                 bool           `json:"can_comment_on_threads"`
	CanDeleteAnyFile                    bool           `json:"can_delete_any_file"`
	CanRemovePeopleFromOrganization     bool           `json:"can_remove_people_from_organization"`
	CanManageGeneralInviteLink          bool           `json:"can_manage_general_invite_link"`
}

func (req *UpdateOrgPermissionsRequest) GetPermissionList() PermissionList {
	if req.PermissionList != (PermissionList{}) {
		return req.PermissionList
	}
	if req.Permissions != (PermissionList{}) {
		return req.Permissions
	}
	return PermissionList{
		CanManageChannels:               req.CanManageChannels,
		CanManageMembers:                req.CanManageMembers,
		CanManageOrganization:           req.CanManageOrganization,
		CanManageSettings:               req.CanManageSettings,
		CanManageBilling:                req.CanManageBilling,
		CanManageAgents:                 req.CanManageAgents,
		CanManageWorkflows:              req.CanManageWorkflows,
		CanManageIntegrations:           req.CanManageIntegrations,
		CanManageSecurity:               req.CanManageSecurity,
		CanManageRoles:                  req.CanManageRoles,
		CanViewAnalytics:                req.CanViewAnalytics,
		CanViewBilling:                  req.CanViewBilling,
		CanViewChannels:                 req.CanViewChannels,
		CanEditMessages:                 req.CanEditMessages,
		CanDeleteMessages:               req.CanDeleteMessages,
		CanDeleteFiles:                  req.CanDeleteFiles,
		CanCreateChannels:               req.CanCreateChannels,
		CanCreateAgents:                 req.CanCreateAgents,
		CanCreateRole:                   req.CanCreateRole,
		CanCreateWebhooks:               req.CanCreateWebhooks,
		CanArchiveChannels:              req.CanArchiveChannels,
		CanInviteMembers:                req.CanInviteMembers,
		CanRemovePeople:                 req.CanRemovePeople,
		CanCommentThreads:               req.CanCommentThreads,
		CanChangeUserOrgRole:            req.CanChangeUserOrgRole,
		CanCreateCustomRole:             req.CanCreateCustomRole,
		CanCreateChannel:                req.CanCreateChannel,
		CanCommentOnThreads:             req.CanCommentOnThreads,
		CanDeleteAnyFile:                req.CanDeleteAnyFile,
		CanRemovePeopleFromOrganization: req.CanRemovePeopleFromOrganization,
		CanManageGeneralInviteLink:      req.CanManageGeneralInviteLink,
	}
}

func (r *OrgRole) CreateOrgRole(db *gorm.DB) error {

	permissionList := r.Permissions.PermissionList
	if permissionList == (PermissionList{}) {
		permissionList = GetUserDefaultPermissions()
	}

	permission := Permission{
		ID:             utility.GenerateUUID(),
		RoleID:         r.ID,
		PermissionList: permissionList,
		IsDefault:      false,
	}

	err := postgresql.CreateOneRecord(db, &r)
	if err != nil {
		return err
	}
	err = postgresql.CreateOneRecord(db, &permission)
	if err != nil {
		return err
	}

	r.Permissions = permission

	return nil
}

func (r *OrgRole) DeleteOrgRole(db *gorm.DB) error {

	if err := postgresql.HardDeleteRecordFromDb(db, r); err != nil {
		return err
	}

	return nil
}

func (r *OrgRole) UpdateOrgRole(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &r)
	return err
}

func (rp *Permission) UpdateOrgPermissions(db *gorm.DB) error {
	_, err := postgresql.SaveAllFields(db, &rp)
	return err
}

func (og *OrgRole) CheckExists(db *gorm.DB, roleID string) bool {
	var o OrgRole
	return postgresql.CheckExists(db, &o, "id = ?", roleID)
}

func (r *OrgRole) GetOrgRoles(db *gorm.DB, orgID string) ([]OrgRole, error) {
	var orgRoles []OrgRole

	query := db.Where("organisation_id = ? OR is_default = ?", orgID, true)
	query = postgresql.PreloadEntities(query, &orgRoles, "Permissions")
	err := query.Find(&orgRoles).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return orgRoles, nil
		}
		return orgRoles, err
	}

	return orgRoles, nil
}

func (r *OrgRole) GetAOrgRole(db *gorm.DB, orgID, roleID string) (OrgRole, error) {
	var orgRole OrgRole

	query := db.Where("organisation_id = ? OR is_default = ?", orgID, true).Where("id = ?", roleID)
	query = postgresql.PreloadEntities(query, &orgRole, "Permissions")

	err := query.First(&orgRole).Error

	if err != nil {
		return orgRole, err
	}

	return orgRole, nil
}

func (r *OrgRole) GetAOrgRoleById(db *gorm.DB, roleID string) (OrgRole, error) {
	var orgRole OrgRole

	query := db.Where("id = ?", roleID)
	query = postgresql.PreloadEntities(query, &orgRole, "Permissions")

	err := query.First(&orgRole).Error

	if err != nil {
		return orgRole, fmt.Errorf("role with id %s not found: %w", roleID, err)
	}

	return orgRole, nil
}

func (r *OrgRole) GetAOrgRoleByName(db *gorm.DB, roleName string) (OrgRole, error) {
	var orgRole OrgRole

	query := db.Where("LOWER(name) = LOWER(?)", roleName)
	err := query.First(&orgRole).Error

	if err != nil {
		return orgRole, fmt.Errorf("role with name %s not found: %w", roleName, err)
	}

	return orgRole, nil
}

func GetRoleName(roleId RoleId) RoleName {
	switch roleId {
	case RoleIdentity.User:
		return UserRoleName
	case RoleIdentity.SuperAdmin:
		return AdminRoleName
	default:
		return "unknown"
	}
}

func (r *OrgRole) UpdateUserRole(db *gorm.DB, userId, orgId, roleId string, c *gin.Context) (*User, error) {
	var (
		user        User
		orgRole     OrgRole
		accessToken AccessToken
		orgMgt      OrgUserManagement
	)

	user, err := user.GetUserByID(db, userId)
	if err != nil {
		return nil, err
	}

	orgRole, err = orgRole.GetAOrgRoleById(db, roleId)
	if err != nil {
		return nil, err
	}

	user.OrgRoleID = &roleId
	user.OrgRole = orgRole

	orgMgt, err = orgMgt.GetByIDs(db, userId, orgId)
	if err != nil {
		return nil, err
	}

	orgMgt.RoleID = roleId
	accessToken, err = user.GetLatestAccessTokenByUserID(db, userId)
	if err != nil {
		return nil, err
	}

	if err := accessToken.RevokeAccessTokenDelete(db); err != nil {
		return nil, fmt.Errorf("error revoking user session: %v", err)
	}

	if _, err := postgresql.SaveAllFields(db, &user); err != nil {
		return nil, err
	}

	if _, err := postgresql.SaveAllFields(db, &orgMgt); err != nil {
		return nil, err
	}

	return &user, nil
}

func (u *UserRole) UpdateUserIdentity(db *gorm.DB, userId string, roleId string) (*User, error) {
	var user User

	user, err := user.GetUserByID(db, userId)
	if err != nil {
		return nil, err
	}

	user.UserRoleID = &roleId

	if _, err := postgresql.SaveAllFields(db, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
