package admin

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	telexaudit "github.com/hngprojects/telex_be/services/telexAudit"
	"github.com/hngprojects/telex_be/utility"
)

type AdminRoleInitiateResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Target    string    `json:"target"`
}

func InitiateAdminRoleChange(db *storage.Database, targetID, newRole, requesterID, ipAdress string) (*AdminRoleInitiateResponse, error) {
	target, err := models.GetAdminById(db.Postgresql, targetID)
	if err != nil {
		return nil, err
	}

	requester, err := models.GetAdminById(db.Postgresql, requesterID)
	if err != nil {
		return nil, err
	}

	if target.Role == newRole {
		return nil, errors.New("admin already has this role")
	}

	// 32-byte secure token
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	confirmation := models.RoleChangeConfirmation{
		ID:                utility.GenerateUUID(),
		TargetAdminID:     target.ID,
		TargetAdminEmail:  target.Email,
		TargetAdminName:   target.Name,
		RequesterID:       requester.ID,
		RequesterEmail:    requester.Email,
		NewRole:           newRole,
		OldRole:           target.Role,
		ConfirmationToken: token,
		ExpiresAt:         time.Now().Add(15 * time.Minute),
		IPAddress:         ipAdress,
	}

	if err := db.Postgresql.Create(&confirmation).Error; err != nil {
		return nil, err
	}

	return &AdminRoleInitiateResponse{
		Token:     token,
		ExpiresAt: confirmation.ExpiresAt,
		Target:    target.Email,
	}, nil
}

func ConfirmAdminRoleChange(db *storage.Database, logger *utility.Logger, token, requesterID string, ipAddress string, userAgent string) error {
	var confirmation models.RoleChangeConfirmation
	if err := db.Postgresql.Where("confirmation_token = ? AND is_used = ?", token, false).First(&confirmation).Error; err != nil {
		return errors.New("invalid or expired token")
	}

	if time.Now().After(confirmation.ExpiresAt) {
		return errors.New("token expired")
	}

	if confirmation.RequesterID != requesterID {
		return errors.New("unauthorized requester")
	}

	var requester models.Admin
	if err := db.Postgresql.Where("id = ?", requesterID).First(&requester).Error; err != nil {
		return errors.New("requester not found")
	}

	var targetAdmin models.Admin
	if err := db.Postgresql.Where("id = ?", confirmation.TargetAdminID).First(&targetAdmin).Error; err != nil {
		return errors.New("target admin not found")
	}

	err := db.Postgresql.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Admin{}).
			Where("id = ?", confirmation.TargetAdminID).
			UpdateColumn("role", confirmation.NewRole).Error; err != nil {
			return fmt.Errorf("failed to change role: %w", err)
		}

		if err := tx.Model(&models.AccessToken{}).
			Where("owner_id = ?", confirmation.TargetAdminID).
			Update("is_live", false).Error; err != nil {
			return fmt.Errorf("failed to invalidate access tokens: %w", err)
		}

		oldValJSON, _ := json.Marshal(map[string]string{"role": confirmation.OldRole})
		newValJSON, _ := json.Marshal(map[string]string{"role": confirmation.NewRole})

		audit := models.AuditLog{
			ID:           utility.GenerateUUID(),
			ActorID:      requesterID,
			ActorEmail:   requester.Email,
			Action:       models.ActionAdminUpdate,
			ResourceID:   confirmation.TargetAdminID,
			ResourceType: models.ResourceAdmin,
			OldValues:    string(oldValJSON),
			NewValues:    string(newValJSON),
			Description:  fmt.Sprintf("Superadmin %s changed role of %s from %s to %s", requester.Email, confirmation.TargetAdminEmail, confirmation.OldRole, confirmation.NewRole),
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
		}

		if err := tx.Create(&audit).Error; err != nil {
			logger.Error("failed to create audit log: " + err.Error())
			return fmt.Errorf("failed to create audit log (transaction rolled back): %w", err)
		}

		if err := tx.Model(&confirmation).Updates(map[string]any{
			"is_used": true,
			"used_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("failed to mark token as used: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	telexaudit.RoleChangeAudit(db, logger, requester.Email, confirmation.TargetAdminEmail, confirmation.OldRole, confirmation.NewRole)

	return nil
}

func GetPlans(db *gorm.DB) ([]models.Plan, error) {
	var plans []models.Plan
	if err := db.Find(&plans).Error; err != nil {
		return nil, err
	}
	return plans, nil
}
