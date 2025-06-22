package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Admin struct {
	ID        string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	Name      string    `gorm:"column:name; type:varchar(255)" json:"name"`
	Email     string    `gorm:"column:email; type:varchar(255)" json:"email"`
	IsActive  bool      `gorm:"column:is_active; type:bool; default:false" json:"is_active"`
	IsDeleted bool      `gorm:"column:is_deleted; type:bool; default:false" json:"is_deleted"`
	Password  string    `gorm:"column:password; type:text; not null" json:"-"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	Role      string    `gorm:"column:role;type:varchar(50);default:'admin'" json:"role"`
}

type AdminLoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CreateAdminRequest struct {
	Email string `json:"email" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Role  string `json:"role"`
}

type CreateAdminResponse struct {
	ID    string `json:"id" validate:"required"`
	Email string `json:"email" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Role  string `json:"role"`
}

const (
	RoleAdmin      = "admin"
	RoleSuperAdmin = "superadmin"
)

func (a *Admin) BeforeSave(tx *gorm.DB) (err error) {
	if a.Role != RoleAdmin && a.Role != RoleSuperAdmin {
		return fmt.Errorf("invalid role: %s", a.Role)
	}
	return nil
}

func (a *Admin) CreateAdmin(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &a)
	if err != nil {
		return err
	}
	return nil
}

func DeleteAdmin(db *gorm.DB, adminID string) error {
	var admin Admin
	if err := db.First(&admin, "id = ?", adminID).Error; err != nil {
		return fmt.Errorf("admin with this ID does not exist")
	}

	if err := db.Delete(&admin).Error; err != nil {
		return err
	}

	return nil
}

func (user *Admin) ActivateAdmin(db *gorm.DB, is_active bool, adminId string) error {
	result := db.Model(&Admin{}).
		Where("id = ?", adminId).
		Update("is_active", is_active)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to activate an Admin")
	}

	return nil
}

func (user *Admin) ChangeRole(db *gorm.DB, role string, adminId string) error {
	if role != "admin" && role != "superadmin" {
		return fmt.Errorf("invalid role: %s", role)
	}

	result := db.Model(&Admin{}).
		Where("id = ?", adminId).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("failed to change role")
	}

	return nil
}

func GetAllAdmins(db *gorm.DB) ([]CreateAdminResponse, error) {
	var admins []Admin
	if err := db.Find(&admins).Error; err != nil {
		return nil, err
	}

	var response []CreateAdminResponse
	for _, admin := range admins {
		response = append(response, CreateAdminResponse{
			ID:    admin.ID,
			Email: admin.Email,
			Name:  admin.Name,
			Role:  admin.Role,
		})
	}

	return response, nil
}
