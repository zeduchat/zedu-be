package admin

import (
	"context"
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
	"github.com/hngprojects/telex_be/utility/audit_utility"
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

		if err := audit_utility.CreateAuditLog(tx, audit_utility.AuditLogParams{
			ActorID:      requesterID,
			ActorEmail:   requester.Email,
			ActorRole:    requester.Role,
			Action:       models.ActionAdminRoleUpdate,
			ResourceType: models.ResourceAdmin,
			ResourceID:   confirmation.TargetAdminID,
			OldValues:    string(oldValJSON),
			NewValues:    string(newValJSON),
			Description:  fmt.Sprintf("Superadmin %s changed role of %s from %s to %s", requester.Email, confirmation.TargetAdminEmail, confirmation.OldRole, confirmation.NewRole),
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			Success:      true,
		}); err != nil {
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

func GetAICreditPackages(db *gorm.DB) ([]models.AICreditPackage, error) {
	var packages []models.AICreditPackage
	if err := db.Find(&packages).Error; err != nil {
		return nil, err
	}
	return packages, nil
}

type PlansResponse struct {
	Plans            []models.Plan            `json:"plans"`
	AICreditPackages []models.AICreditPackage `json:"ai_credit_packages"`
}

func GetPlansWithPackages(db *gorm.DB) (*PlansResponse, error) {
	plans, err := GetPlans(db)
	if err != nil {
		return nil, err
	}

	packages, err := GetAICreditPackages(db)
	if err != nil {
		return nil, err
	}

	return &PlansResponse{
		Plans:            plans,
		AICreditPackages: packages,
	}, nil
}

// BroadcastNotificationToAllUsers sends a broadcast notification to all users via OneSignal
func BroadcastNotificationToAllUsers(db *storage.Database, logger *utility.Logger,
	adminID, adminEmail, ipAddress, userAgent string,
	req models.BroadcastNotificationRequest,
) (*models.BroadcastNotificationResponse, error) {
	// Create initial broadcast log with status "started"
	broadcastLogID := utility.GenerateUUID()
	broadcastLog := models.BroadcastNotificationLog{
		ID:                 broadcastLogID,
		AdminID:            adminID,
		AdminEmail:         adminEmail,
		Title:              req.Title,
		Message:            req.Message,
		AvatarUrl:          req.AvatarUrl,
		TotalUsersTargeted: 0, // Will be updated by the worker
		SuccessfullySent:   0,
		FailedCount:        0,
		Status:             "started",
		ScheduledAt:        req.ScheduledAt,
		IPAddress:          ipAddress,
		UserAgent:          userAgent,
	}

	if err := db.Postgresql.Create(&broadcastLog).Error; err != nil {
		logger.Error("Failed to create broadcast notification log: %s", err.Error())
		return nil, fmt.Errorf("failed to create broadcast log: %w", err)
	}

	// Enqueue the broadcast notification job
	jobArgs := models.BroadcastNotificationJobArgs{
		BroadcastLogID: broadcastLogID,
		AdminID:        adminID,
		AdminEmail:     adminEmail,
		Title:          req.Title,
		Message:        req.Message,
		AvatarUrl:      req.AvatarUrl,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	}

	_, err := jobArgs.InsertBroadcastNotificationJob(context.Background(), db)
	if err != nil {
		logger.Error("Failed to enqueue broadcast notification job: %s", err.Error())
		// Update status to failed
		db.Postgresql.Model(&models.BroadcastNotificationLog{}).
			Where("id = ?", broadcastLogID).
			Update("status", "failed")
		return nil, fmt.Errorf("failed to enqueue broadcast job: %w", err)
	}

	// Create audit log entry
	description := fmt.Sprintf("Admin %s sent broadcast notification to all users", adminEmail)
	newValJSON, _ := json.Marshal(map[string]any{
		"title":   req.Title,
		"message": req.Message,
		"status":  "started",
	})

	if auditErr := audit_utility.CreateAuditLog(db.Postgresql, audit_utility.AuditLogParams{
		ActorID:      adminID,
		ActorEmail:   adminEmail,
		ActorRole:    "admin",
		Action:       models.ActionBroadcastNotificationCreated,
		ResourceType: models.ResourceNotification,
		ResourceID:   broadcastLogID,
		NewValues:    string(newValJSON),
		Description:  description,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Success:      true,
	}); auditErr != nil {
		logger.Error("Failed to create audit log for broadcast notification: %v", auditErr)
	}

	return &models.BroadcastNotificationResponse{
		ID:                 broadcastLogID,
		Title:              req.Title,
		Message:            req.Message,
		TotalUsersTargeted: 0, // Will be updated by worker
		SuccessfullySent:   0,
		CreatedAt:          broadcastLog.CreatedAt,
		ScheduledAt:        req.ScheduledAt,
	}, nil
}

// GetBroadcastNotificationStatus retrieves the status of a broadcast notification by ID
func GetBroadcastNotificationStatus(db *storage.Database, broadcastID string) (*models.BroadcastNotificationLog, error) {
	var broadcastLog models.BroadcastNotificationLog
	if err := db.Postgresql.Where("id = ?", broadcastID).First(&broadcastLog).Error; err != nil {
		return nil, err
	}
	return &broadcastLog, nil
}
