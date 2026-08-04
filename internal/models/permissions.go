package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Permission name constants for use with PermissionMiddleware and userCanOrOwner.
const (
	PermManageChannels               = "can_manage_channels"
	PermManageMembers                = "can_manage_members"
	PermManageOrganization           = "can_manage_organization"
	PermManageSettings               = "can_manage_settings"
	PermManageBilling                = "can_manage_billing"
	PermManageAgents                 = "can_manage_agents"
	PermManageWorkflows              = "can_manage_workflows"
	PermManageIntegrations           = "can_manage_integrations"
	PermManageSecurity               = "can_manage_security"
	PermManageRoles                  = "can_manage_roles"
	PermViewAnalytics                = "can_view_analytics"
	PermViewBilling                  = "can_view_billing"
	PermViewChannels                 = "can_view_channels"
	PermEditMessages                 = "can_edit_messages"
	PermDeleteMessages               = "can_delete_messages"
	PermDeleteFiles                  = "can_delete_files"
	PermCreateChannels               = "can_create_channels"
	PermCreateAgents                 = "can_create_agents"
	PermCreateRole                   = "can_create_role"
	PermCreateWebhooks               = "can_create_webhooks"
	PermArchiveChannels              = "can_archive_channels"
	PermInviteMembers                = "can_invite_members"
	PermRemovePeople                 = "can_remove_people"
	PermCommentThreads               = "can_comment_threads"
	PermChangeUserOrgRole            = "can_change_user_org_role"
	PermRemovePeopleFromOrganization = "can_remove_people_from_organization"
	PermCreateCustomRole             = "can_create_custom_role"
	PermCreateChannel                = "can_create_channel"
	PermCommentOnThreads             = "can_comment_on_threads"
	PermDeleteAnyFile                = "can_delete_any_file"
	PermManageGeneralInviteLink      = "can_manage_general_invite_link"
)

type Permission struct {
	ID             string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	RoleID         string         `gorm:"unique;null" json:"role_id"`
	PermissionList PermissionList `gorm:"type:jsonb" json:"permission_list"`
	IsDefault      bool           `gorm:"type:bool" json:"is_default"`
	CreatedAt      time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"-"`
	UpdatedAt      time.Time      `gorm:"column:updated_at; null; autoUpdateTime" json:"-"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type PermissionList struct {
	CanManageChannels     bool `json:"can_manage_channels,omitempty"`
	CanManageMembers      bool `json:"can_manage_members,omitempty"`
	CanManageOrganization bool `json:"can_manage_organization,omitempty"`
	CanManageSettings     bool `json:"can_manage_settings,omitempty"`
	CanManageBilling      bool `json:"can_manage_billing,omitempty"`
	CanManageAgents       bool `json:"can_manage_agents,omitempty"`
	CanManageWorkflows    bool `json:"can_manage_workflows,omitempty"`
	CanManageIntegrations bool `json:"can_manage_integrations,omitempty"`
	CanManageSecurity     bool `json:"can_manage_security,omitempty"`
	CanManageRoles        bool `json:"can_manage_roles,omitempty"`
	CanViewAnalytics      bool `json:"can_view_analytics,omitempty"`
	CanViewBilling        bool `json:"can_view_billing,omitempty"`
	CanViewChannels       bool `json:"can_view_channels,omitempty"`
	CanEditMessages       bool `json:"can_edit_messages,omitempty"`
	CanDeleteMessages     bool `json:"can_delete_messages,omitempty"`
	CanDeleteFiles        bool `json:"can_delete_files,omitempty"`
	CanCreateChannels     bool `json:"can_create_channels,omitempty"`
	CanCreateAgents       bool `json:"can_create_agents,omitempty"`
	CanCreateRole         bool `json:"can_create_role,omitempty"`
	CanCreateWebhooks     bool `json:"can_create_webhooks,omitempty"`
	CanArchiveChannels    bool `json:"can_archive_channels,omitempty"`
	CanInviteMembers      bool `json:"can_invite_members,omitempty"`
	CanRemovePeople       bool `json:"can_remove_people,omitempty"`
	CanCommentThreads     bool `json:"can_comment_threads,omitempty"`
	CanChangeUserOrgRole  bool `json:"can_change_user_org_role,omitempty"`

	// Legacy field aliases for backward compatibility with older records
	CanRemovePeopleFromOrganization bool `json:"can_remove_people_from_organization,omitempty"`
	CanCreateCustomRole             bool `json:"can_create_custom_role,omitempty"`
	CanCreateChannel                bool `json:"can_create_channel,omitempty"`
	CanCommentOnThreads             bool `json:"can_comment_on_threads,omitempty"`
	CanDeleteAnyFile                bool `json:"can_delete_any_file,omitempty"`
	CanManageGeneralInviteLink      bool `json:"can_manage_general_invite_link,omitempty"`
}

type SystemPermissionInfo struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

func GetMasterSystemPermissions() []SystemPermissionInfo {
	return []SystemPermissionInfo{
		{Key: PermManageChannels, Name: "Manage Channels", Category: "channels", Description: "Full management over channels"},
		{Key: PermManageMembers, Name: "Manage Members", Category: "members", Description: "Manage organisation members"},
		{Key: PermManageOrganization, Name: "Manage Organisation", Category: "organization", Description: "Manage organisation settings & profile"},
		{Key: PermManageSettings, Name: "Manage Settings", Category: "settings", Description: "Manage workspace settings"},
		{Key: PermManageBilling, Name: "Manage Billing", Category: "billing", Description: "Manage subscription & billing"},
		{Key: PermManageAgents, Name: "Manage Agents", Category: "agents", Description: "Manage AI agents and bots"},
		{Key: PermManageWorkflows, Name: "Manage Workflows", Category: "workflows", Description: "Manage automated workflows"},
		{Key: PermManageIntegrations, Name: "Manage Integrations", Category: "integrations", Description: "Manage external integrations"},
		{Key: PermManageSecurity, Name: "Manage Security", Category: "security", Description: "Manage security policies & audit logs"},
		{Key: PermManageRoles, Name: "Manage Roles", Category: "roles", Description: "Manage custom roles & permissions"},
		{Key: PermViewAnalytics, Name: "View Analytics", Category: "analytics", Description: "View workspace analytics & metrics"},
		{Key: PermViewBilling, Name: "View Billing", Category: "billing", Description: "View billing details"},
		{Key: PermViewChannels, Name: "View Channels", Category: "channels", Description: "View organisation channels"},
		{Key: PermEditMessages, Name: "Edit Messages", Category: "messages", Description: "Edit posted messages"},
		{Key: PermDeleteMessages, Name: "Delete Messages", Category: "messages", Description: "Delete messages"},
		{Key: PermDeleteFiles, Name: "Delete Files", Category: "files", Description: "Delete uploaded files"},
		{Key: PermCreateChannels, Name: "Create Channels", Category: "channels", Description: "Create new channels"},
		{Key: PermCreateAgents, Name: "Create Agents", Category: "agents", Description: "Create new AI agents"},
		{Key: PermCreateRole, Name: "Create Role", Category: "roles", Description: "Create custom roles"},
		{Key: PermCreateWebhooks, Name: "Create Webhooks", Category: "webhooks", Description: "Create webhooks & triggers"},
		{Key: PermArchiveChannels, Name: "Archive Channels", Category: "channels", Description: "Archive and unarchive channels"},
		{Key: PermInviteMembers, Name: "Invite Members", Category: "members", Description: "Send invitations to new members"},
		{Key: PermRemovePeople, Name: "Remove People", Category: "members", Description: "Remove members from organisation"},
		{Key: PermCommentThreads, Name: "Comment Threads", Category: "threads", Description: "Post messages & thread comments"},
		{Key: PermChangeUserOrgRole, Name: "Change User Role", Category: "roles", Description: "Update user organisation roles"},
	}
}

func GetUserDefaultPermissions() PermissionList {
	return PermissionList{
		CanViewChannels:   true,
		CanEditMessages:   true,
		CanCreateChannels: true,
		CanInviteMembers:  true,
		CanCommentThreads: true,
	}
}

func (p PermissionList) ToMap() map[string]bool {
	manageChannels := p.CanManageChannels
	manageMembers := p.CanManageMembers
	manageOrg := p.CanManageOrganization || p.CanManageGeneralInviteLink
	manageSettings := p.CanManageSettings
	manageBilling := p.CanManageBilling
	manageAgents := p.CanManageAgents
	manageWorkflows := p.CanManageWorkflows
	manageIntegrations := p.CanManageIntegrations
	manageSecurity := p.CanManageSecurity
	manageRoles := p.CanManageRoles
	viewAnalytics := p.CanViewAnalytics
	viewBilling := p.CanViewBilling
	viewChannels := p.CanViewChannels
	editMessages := p.CanEditMessages
	deleteMessages := p.CanDeleteMessages
	deleteFiles := p.CanDeleteFiles || p.CanDeleteAnyFile
	createChannels := p.CanCreateChannels || p.CanCreateChannel
	createAgents := p.CanCreateAgents
	createRole := p.CanCreateRole || p.CanCreateCustomRole
	createWebhooks := p.CanCreateWebhooks
	archiveChannels := p.CanArchiveChannels
	inviteMembers := p.CanInviteMembers
	removePeople := p.CanRemovePeople || p.CanRemovePeopleFromOrganization
	commentThreads := p.CanCommentThreads || p.CanCommentOnThreads
	changeUserRole := p.CanChangeUserOrgRole

	return map[string]bool{
		"can_manage_channels":      manageChannels,
		"can_manage_members":       manageMembers,
		"can_manage_organization":  manageOrg,
		"can_manage_settings":      manageSettings,
		"can_manage_billing":       manageBilling,
		"can_manage_agents":        manageAgents,
		"can_manage_workflows":     manageWorkflows,
		"can_manage_integrations":  manageIntegrations,
		"can_manage_security":      manageSecurity,
		"can_manage_roles":         manageRoles,
		"can_view_analytics":       viewAnalytics,
		"can_view_billing":         viewBilling,
		"can_view_channels":        viewChannels,
		"can_edit_messages":        editMessages,
		"can_delete_messages":      deleteMessages,
		"can_delete_files":         deleteFiles,
		"can_create_channels":      createChannels,
		"can_create_agents":        createAgents,
		"can_create_role":          createRole,
		"can_create_webhooks":      createWebhooks,
		"can_archive_channels":     archiveChannels,
		"can_invite_members":       inviteMembers,
		"can_remove_people":        removePeople,
		"can_comment_threads":      commentThreads,
		"can_change_user_org_role": changeUserRole,
	}
}

func (p PermissionList) ToSlice() []string {
	var permissions []string
	for perm, enabled := range p.ToMap() {
		if enabled {
			permissions = append(permissions, perm)
		}
	}
	return permissions
}

func (p *PermissionList) Scan(value any) error {
	if b, ok := value.([]byte); ok {
		return json.Unmarshal(b, &p)
	}
	return fmt.Errorf("type assertion to []byte failed")
}

func (p PermissionList) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *Permission) GetAOrgPermission(db *gorm.DB, roleID string) (Permission, error) {
	var perm Permission
	query := db.Where("role_id = ?", roleID)
	err := query.First(&perm).Error

	if err != nil {
		return perm, err
	}

	return perm, nil
}

func OrgUserHasPermission(permissionList PermissionList, permission string) bool {
	return permissionList.ToMap()[permission]
}
