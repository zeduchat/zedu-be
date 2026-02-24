package fileManagement

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	storage "github.com/hngprojects/telex_be/pkg/repository/storage"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	"github.com/hngprojects/telex_be/utility"
)

var (
	ErrFileShareNotFound    = errors.New("file share not found")
	ErrShareExpired         = errors.New("file share has expired")
	ErrAccessDenied         = errors.New("access denied to this file")
	ErrInvalidShareLink     = errors.New("invalid share link")
	ErrCannotShareOwnFile   = errors.New("cannot share file with yourself")
	ErrUserNotInOrg         = errors.New("user is not in same organization")
	ErrFileNotFound         = errors.New("file not found")
	ErrEditPermissionDenied = errors.New("user does not have edit permission")
)

type CreateFileShareParams struct {
	FileID         string
	SharedByUserID string
	OrganisationID string
	AccessType     string
	PermissionType string
	Note           string
	ExpiresAt      *time.Time
}

// CreateFileShare creates a new file share record with a unique share link
func CreateFileShare(db *gorm.DB, logger *utility.Logger, params CreateFileShareParams) (*models.FileShare, error) {
	// Validate file exists
	var file models.File
	if err := db.Where("id = ?", params.FileID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("file not found")
		}
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}

	// Validate access type
	if err := models.ValidateAccessType(params.AccessType); err != nil {
		return nil, err
	}

	// Validate permission type
	if err := models.ValidatePermissionType(params.PermissionType); err != nil {
		return nil, err
	}

	// Validate expiration
	if err := models.ValidateShareExpiration(params.ExpiresAt); err != nil {
		return nil, err
	}

	// Generate unique share link
	shareLink, err := generateUniqueShareLink(db)
	if err != nil {
		return nil, fmt.Errorf("failed to generate share link: %w", err)
	}

	// Create file share record
	now := time.Now().UTC()
	fileShare := &models.FileShare{
		ID:             utility.GenerateUUID(),
		FileID:         params.FileID,
		SharedByUserID: params.SharedByUserID,
		OrganisationID: params.OrganisationID,
		AccessType:     params.AccessType,
		PermissionType: params.PermissionType,
		Note:           params.Note,
		ShareLink:      shareLink,
		ExpiresAt:      params.ExpiresAt,
		AccessCount:    0,
		LastAccessedAt: nil,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := db.Create(fileShare).Error; err != nil {
		return nil, fmt.Errorf("failed to create file share: %w", err)
	}

	logger.Info("File share created successfully", "file_share_id", fileShare.ID, "file_id", params.FileID)
	return fileShare, nil
}

// generateUniqueShareLink generates a unique share link
func generateUniqueShareLink(db *gorm.DB) (string, error) {
	maxAttempts := 5

	for i := 0; i < maxAttempts; i++ {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		token := hex.EncodeToString(bytes)
		shareLink := fmt.Sprintf("%s/s/%s", config.Config.App.Url, token)

		var count int64
		if err := db.Model(&models.FileShare{}).Where("share_link = ?", shareLink).Count(&count).Error; err != nil {
			return "", err
		}

		if count == 0 {
			return shareLink, nil
		}
	}

	return "", errors.New("failed to generate unique share link after multiple attempts")
}

// UpdateFileShare updates an existing file share
func UpdateFileShare(db *gorm.DB, logger *utility.Logger, shareID string, req models.UpdateFileShareRequest) (*models.FileShare, error) {
	var fileShare models.FileShare

	if err := db.Where("id = ?", shareID).First(&fileShare).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileShareNotFound
		}
		return nil, fmt.Errorf("failed to fetch file share: %w", err)
	}

	// Validate and update fields
	if req.AccessType != nil {
		if err := models.ValidateAccessType(*req.AccessType); err != nil {
			return nil, err
		}
		fileShare.AccessType = *req.AccessType
	}

	if req.PermissionType != nil {
		if err := models.ValidatePermissionType(*req.PermissionType); err != nil {
			return nil, err
		}
		fileShare.PermissionType = *req.PermissionType
	}

	if req.Note != nil {
		fileShare.Note = *req.Note
	}

	if req.ExpiresAt != nil {
		if err := models.ValidateShareExpiration(req.ExpiresAt); err != nil {
			return nil, err
		}
		fileShare.ExpiresAt = req.ExpiresAt
	}

	fileShare.UpdatedAt = time.Now().UTC()

	if err := db.Save(&fileShare).Error; err != nil {
		return nil, fmt.Errorf("failed to update file share: %w", err)
	}

	logger.Info("File share updated successfully", "file_share_id", shareID)
	return &fileShare, nil
}

// GetFileShareByLink retrieves file share by share link and validates it
func GetFileShareByLink(db *gorm.DB, shareLink string) (*models.FileShare, error) {
	var fileShare models.FileShare

	if err := db.Where("share_link = ?", shareLink).First(&fileShare).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileShareNotFound
		}
		return nil, fmt.Errorf("failed to fetch file share: %w", err)
	}

	// Check if share is expired
	if fileShare.ExpiresAt != nil && fileShare.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrShareExpired
	}

	return &fileShare, nil
}

// GetFileSharesForFile retrieves all shares for a specific file
func GetFileSharesForFile(db *gorm.DB, fileID string, activeOnly bool) ([]models.FileShare, error) {
	var shares []models.FileShare

	query := db.Where("file_id = ?", fileID)

	if activeOnly {
		query = query.Where("(expires_at IS NULL OR expires_at > ?)", time.Now().UTC())
	}

	if err := query.Order("created_at DESC").Find(&shares).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch file shares: %w", err)
	}

	return shares, nil
}

// RevokeFileShare revokes/deletes a file share
func RevokeFileShare(db *gorm.DB, logger *utility.Logger, shareID, requestingUserID string) error {
	var fileShare models.FileShare

	if err := db.Where("id = ?", shareID).First(&fileShare).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFileShareNotFound
		}
		return fmt.Errorf("failed to fetch file share: %w", err)
	}

	// Only creator can revoke
	if fileShare.SharedByUserID != requestingUserID {
		return ErrAccessDenied
	}

	if err := db.Delete(&fileShare).Error; err != nil {
		return fmt.Errorf("failed to revoke file share: %w", err)
	}

	logger.Info("File share revoked successfully", "file_share_id", shareID)
	return nil
}

// IncrementShareAccess tracks when a shared file is accessed
func IncrementShareAccess(db *gorm.DB, logger *utility.Logger, shareID string) error {
	now := time.Now().UTC()

	if err := db.Model(&models.FileShare{}).
		Where("id = ?", shareID).
		Updates(map[string]interface{}{
			"access_count":     gorm.Expr("access_count + 1"),
			"last_accessed_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to update share access count: %w", err)
	}

	return nil
}

// CheckFileAccess determines if a user can access a file
func CheckFileAccess(db *gorm.DB, fileID, userID, orgID string) (bool, string, error) {
	var file models.File

	// First check if file exists
	if err := db.Where("id = ?", fileID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "file_not_found", ErrFileShareNotFound
		}
		return false, "database_error", err
	}

	// Check if user owns the file
	if file.UserID == userID {
		return true, "owner", nil
	}

	// Check if file is public within organization
	if file.AccessType == "public" && file.OrganisationID == orgID {
		return true, "public", nil
	}

	// Check if user has explicit share
	var shareCount int64
	if err := db.Model(&models.FileShare{}).
		Where("file_id = ? AND (expires_at IS NULL OR expires_at > ?)", fileID, time.Now().UTC()).
		Count(&shareCount).Error; err != nil {
		return false, "database_error", err
	}

	if shareCount > 0 {
		// Need to verify specific user access through share mechanism
		return true, "shared", nil
	}

	return false, "no_access", ErrAccessDenied
}

// CheckFileEditPermission determines if a user can edit a file
// Returns true if user:
// - Is the file owner, OR
// - Has role permission to delete any file, OR
// - Has at least one active share with edit permission
func CheckFileEditPermission(db *gorm.DB, fileID, userID, orgID string) (bool, error) {
	// Check if file exists (include soft-deleted files for permanent delete operations)
	var file models.File
	if err := db.Unscoped().Where("id = ?", fileID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrFileNotFound
		}
		return false, fmt.Errorf("failed to fetch file: %w", err)
	}

	// Owner can always edit
	if file.UserID == userID {
		return true, nil
	}

	// Check if user has role-based permission to edit/delete any file
	var user models.User
	if err := db.Where("id = ?", userID).First(&user).Error; err == nil {
		if user.OrgRoleID != nil {
			var permission models.Permission
			if err := db.Where("role_id = ?", *user.OrgRoleID).First(&permission).Error; err == nil {
				if permission.PermissionList.CanDeleteAnyFile {
					return true, nil
				}
			}
		}
	}

	// Check for active shares with edit permission
	var shareCount int64
	err := db.Model(&models.FileShare{}).
		Where("file_id = ? AND organisation_id = ?", fileID, orgID).
		Where("permission_type = ?", "edit").
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now().UTC()).
		Count(&shareCount).Error

	if err != nil {
		return false, fmt.Errorf("failed to check file shares: %w", err)
	}

	// User has edit permission if any share grants it (OR logic)
	return shareCount > 0, nil
}

// SendFileToUsersDM sends file to multiple users via DM
func SendFileToUsersDM(db *storage.Database, logger *utility.Logger, extReq request.ExternalRequest,
	fileID string,
	recipientIDs []string,
	senderID, orgID string,
	note string,
	permissionType, accessType, shareLink string) ([]models.DMRecipientInfo, error) {

	// Get file details
	var file models.File
	if err := db.Postgresql.Where("id = ?", fileID).First(&file).Error; err != nil {
		return nil, fmt.Errorf("file not found")
	}

	// Get sender profile
	var senderProfile models.Profile
	if err := db.Postgresql.Where("userid = ?", senderID).First(&senderProfile).Error; err != nil {
		return nil, fmt.Errorf("sender profile not found")
	}

	var recipients []models.DMRecipientInfo
	successfulSends := []string{}

	for _, recipientID := range recipientIDs {
		// Skip sending to self
		if recipientID == senderID {
			recipients = append(recipients, models.DMRecipientInfo{
				UserID:   recipientID,
				Username: senderProfile.UserName,
				Success:  false,
				Error:    "cannot send to yourself",
			})
			continue
		}

		// Verify recipient exists in organization (not deactivated)
		// Users belong to orgs through org_user_managements join table
		var orgUserMgt models.OrgUserManagement
		if err := db.Postgresql.
			Where("user_id = ? AND organisation_id = ? AND is_deactivated = ?", recipientID, orgID, false).
			First(&orgUserMgt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// User either doesn't exist, not in org, or is deactivated
				// Check if user exists at all
				var userCheck models.User
				userExistsErr := db.Postgresql.Where("id = ?", recipientID).First(&userCheck).Error
				if errors.Is(userExistsErr, gorm.ErrRecordNotFound) {
					recipients = append(recipients, models.DMRecipientInfo{
						UserID:  recipientID,
						Success: false,
						Error:   "user not found",
					})
				} else {
					// User exists but not in this org or deactivated
					recipients = append(recipients, models.DMRecipientInfo{
						UserID:  recipientID,
						Success: false,
						Error:   "user not found or not in organization",
					})
				}
			} else {
				recipients = append(recipients, models.DMRecipientInfo{
					UserID:  recipientID,
					Success: false,
					Error:   "failed to verify user membership",
				})
			}
			continue
		}

		// Get recipient user details with profile
		var recipient models.User
		if err := db.Postgresql.Preload("Profile").Where("id = ?", recipientID).First(&recipient).Error; err != nil {
			recipients = append(recipients, models.DMRecipientInfo{
				UserID:  recipientID,
				Success: false,
				Error:   "user profile not found",
			})
			continue
		}

		// Get or create DM channel
		dmChannel := models.DmChannels{
			ID:            utility.GenerateUUID(),
			ChannelId:     utility.GenerateUUID(),
			UserId:        senderID,
			OrgId:         orgID,
			ChatType:      "user",
			ParticipantId: &recipientID,
		}

		dmResponse, err := dmChannel.CreateDmChannel(db.Postgresql)
		if err != nil {
			recipients = append(recipients, models.DMRecipientInfo{
				UserID:   recipientID,
				Username: recipient.Profile.UserName,
				Success:  false,
				Error:    "failed to create DM channel",
			})
			continue
		}

		// Format message content
		content := formatFileShareMessage(senderProfile, file, note, permissionType, accessType, shareLink)

		// Create thread message with file attached
		threadReq := models.CreateThreadMsgReq{
			Content:    content,
			ChannelsID: dmResponse.ID,
			UserId:     senderID,
			OrgId:      orgID,
			Media:      []models.File{file},
			Type:       "file_share",
		}

		threadDoc, _, err := dm.CreateThreadDmMessage(threadReq, db, logger, extReq)
		if err != nil {
			recipients = append(recipients, models.DMRecipientInfo{
				UserID:   recipientID,
				Username: recipient.Profile.UserName,
				Success:  false,
				Error:    "failed to send DM message",
			})
			continue
		}

		successfulSends = append(successfulSends, recipientID)
		recipients = append(recipients, models.DMRecipientInfo{
			UserID:   recipientID,
			Username: recipient.Profile.UserName,
			Success:  true,
		})

		logger.Info("File sent to DM successfully", "file_id", fileID, "recipient", recipientID, "thread_id", threadDoc.ID)
	}

	// Publish notification about successful sends
	if len(successfulSends) > 0 {
		publishFileShareNotification(logger, senderID, orgID, fileID, successfulSends)
	}

	return recipients, nil
}

// formatFileShareMessage creates a message content for file share DM
func formatFileShareMessage(senderProfile models.Profile, file models.File, note, permissionType, accessType, shareLink string) string {
	content := fmt.Sprintf("📎 **File Shared**\n\n")
	content += fmt.Sprintf("%s shared a file **%s** with you.\n\n", senderProfile.UserName, file.FileName)

	if note != "" {
		content += fmt.Sprintf("💬 **Note:** %s\n\n", note)
	}

	content += fmt.Sprintf("🔐 **Permission:** %s\n", permissionType)
	content += fmt.Sprintf("🌐 **Access:** %s\n", accessType)

	if shareLink != "" {
		content += fmt.Sprintf("🔗 **Link:** %s\n\n", shareLink)
	}

	return content
}

// publishFileShareNotification sends real-time notification about file sharing
func publishFileShareNotification(logger *utility.Logger, senderID, orgID, fileID string, recipientIDs []string) {
	for _, recipientID := range recipientIDs {
		notification := models.Notification[models.NotificationType("file_shared")]
		notification.SectionType = models.DmChannelsSection
		notification.Content = map[string]interface{}{
			"file_id":   fileID,
			"shared_by": senderID,
			"event":     "file_shared",
		}
		notification.ModificationDetails = &models.ModificationDetails{
			UserId: recipientID,
			OrgId:  orgID,
		}
		notification.NotificationId = utility.GenerateUUID()

		if err := centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", orgID, recipientID), notification); err != nil {
			logger.Error("Failed to publish file share notification", "recipient_id", recipientID, "error", err)
		}
	}
}

// ShareFileWithUsers combines share creation and DM sending
func ShareFileWithUsers(db *storage.Database, logger *utility.Logger, extReq request.ExternalRequest,
	req models.ShareFileRequest,
	senderID, orgID string) (*models.ShareFileResponse, error) {

	// Validate file ownership
	var file models.File
	if err := db.Postgresql.Where("id = ? AND organisation_id = ?", req.FileID, orgID).First(&file).Error; err != nil {
		return nil, fmt.Errorf("file not found or access denied")
	}

	// Create file share
	shareParams := CreateFileShareParams{
		FileID:         req.FileID,
		SharedByUserID: senderID,
		OrganisationID: orgID,
		AccessType:     req.AccessType,
		PermissionType: req.PermissionType,
		Note:           req.Note,
		ExpiresAt:      req.ExpiresAt,
	}

	fileShare, err := CreateFileShare(db.Postgresql, logger, shareParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create file share: %w", err)
	}

	response := &models.ShareFileResponse{
		FileShareID:    fileShare.ID,
		FileID:         req.FileID,
		ShareLink:      fileShare.ShareLink,
		AccessType:     req.AccessType,
		PermissionType: req.PermissionType,
		Note:           req.Note,
		ExpiresAt:      req.ExpiresAt,
		DMsSent:        []string{},
		RecipientsInfo: []models.DMRecipientInfo{},
	}

	// Send to DMs if requested
	if req.ShareViaDM && len(req.RecipientIDs) > 0 {
		recipients, err := SendFileToUsersDM(
			db,
			logger,
			extReq,
			req.FileID,
			req.RecipientIDs,
			senderID,
			orgID,
			req.Note,
			req.PermissionType,
			req.AccessType,
			fileShare.ShareLink,
		)
		if err != nil {
			logger.Error("Failed to send file to some users via DM", "error", err)
		}

		response.RecipientsInfo = recipients

		// Collect successful recipients
		for _, r := range recipients {
			if r.Success {
				response.DMsSent = append(response.DMsSent, r.UserID)
			}
		}
	}

	return response, nil
}

// UpdateFileAccessSettings updates file-level access settings
func UpdateFileAccessSettings(db *gorm.DB, logger *utility.Logger,
	fileID, userID, orgID string,
	req models.UpdateFileAccessSettingsRequest) (*models.File, error) {

	var file models.File
	if err := db.Where("id = ? AND organisation_id = ?", fileID, orgID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("file not found")
		}
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}

	// Only owner can update access settings
	if file.UserID != userID {
		return nil, ErrAccessDenied
	}

	// Validate and update access type
	if req.AccessType != nil {
		if err := models.ValidateAccessType(*req.AccessType); err != nil {
			return nil, err
		}
		file.AccessType = *req.AccessType
	}

	if req.IsShareable != nil {
		file.IsShareable = *req.IsShareable
	}

	if err := db.Save(&file).Error; err != nil {
		return nil, fmt.Errorf("failed to update file access settings: %w", err)
	}

	logger.Info("File access settings updated", "file_id", fileID, "access_type", file.AccessType)
	return &file, nil
}

// GetFileShareWithDetails retrieves file share with related information
func GetFileShareWithDetails(db *gorm.DB, shareID string) (*models.FileShareListResponse, error) {
	var fileShare models.FileShare

	query := db.Where("id = ?", shareID)

	if err := query.Preload("File").Preload("SharedBy").First(&fileShare).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileShareNotFound
		}
		return nil, fmt.Errorf("failed to fetch file share: %w", err)
	}

	// Check if share is expired
	if fileShare.ExpiresAt != nil && fileShare.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrShareExpired
	}

	response := &models.FileShareListResponse{
		FileShare: &fileShare,
	}

	// Include file info if available
	if fileShare.File != nil {
		response.FileInfo = fileShare.File
	}

	// Include shared by info if available
	if fileShare.SharedBy != nil {
		response.SharedBy = &models.ShareByUser{
			UserID:    fileShare.SharedBy.Userid,
			Username:  fileShare.SharedBy.UserName,
			AvatarURL: fileShare.SharedBy.AvatarURL,
		}
	}

	return response, nil
}
