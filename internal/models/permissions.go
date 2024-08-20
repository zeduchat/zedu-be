package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
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
	CanRemovePeopleFromOrganization bool `json:"can_remove_people_from_organization"`
	CanInviteMembers                bool `json:"can_invite_members"`
	CanCreateCustomRole             bool `json:"can_create_custom_role"`
	CanCreateChannel                bool `json:"can_create_channel"`
	CanCommentOnThreads             bool `json:"can_comment_on_threads"`
	CanViewBilling                  bool `json:"can_view_billing"`
	CanCreateWebhooks               bool `json:"can_create_webhooks"`
	CanViewChannels                 bool `json:"can_view_channels"`
	CanChangeUserOrgRole            bool `json:"can_change_user_org_role"`
}

func (p *PermissionList) Scan(value interface{}) error {
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
	permissionMap := map[string]bool{
		"can_remove_people_from_organization": permissionList.CanRemovePeopleFromOrganization,
		"can_invite_members":                  permissionList.CanInviteMembers,
		"can_create_custom_role":              permissionList.CanCreateCustomRole,
		"can_create_channel":                  permissionList.CanCreateChannel,
		"can_comment_on_threads":              permissionList.CanCommentOnThreads,
		"can_view_billing":                    permissionList.CanViewBilling,
		"can_create_webhooks":                 permissionList.CanCreateWebhooks,
		"can_view_channels":                   permissionList.CanViewChannels,
		"can_change_user_org_role":            permissionList.CanChangeUserOrgRole,
	}

	if allowed, exists := permissionMap[permission]; exists && allowed {
		return true
	}

	return false
}
