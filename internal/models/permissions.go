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
	CanViewTransactions       bool `json:"can_view_transactions"`
	CanViewRefunds            bool `json:"can_view_refunds"`
	CanLogRefund              bool `json:"can_log_refund"`
	CanViewUser               bool `json:"can_view_user"`
	CanEditUser               bool `json:"can_edit_user"`
	CanCreateUser             bool `json:"can_create_user"`
	CanBlacklistWhitelistUser bool `json:"can_blacklist_whitelist_user"`
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

func OrgUserHasAnyPermission(orgRole OrgRole, permissions ...string) bool {
	for _, permission := range permissions {
		switch permission {
		case "can_view_transactions":
			if orgRole.Permissions.PermissionList.CanViewTransactions {
				return true
			}
		case "can_view_refunds":
			if orgRole.Permissions.PermissionList.CanViewRefunds {
				return true
			}
		case "can_log_refund":
			if orgRole.Permissions.PermissionList.CanLogRefund {
				return true
			}
		case "can_view_user":
			if orgRole.Permissions.PermissionList.CanViewUser {
				return true
			}
		case "can_edit_user":
			if orgRole.Permissions.PermissionList.CanEditUser {
				return true
			}
		case "can_create_user":
			if orgRole.Permissions.PermissionList.CanCreateUser {
				return true
			}
		case "can_blacklist_whitelist_user":
			if orgRole.Permissions.PermissionList.CanBlacklistWhitelistUser {
				return true
			}
		default:
			return false
		}
	}
	return false
}
