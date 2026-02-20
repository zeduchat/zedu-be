package utils

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)


func UpdateFilesMetadata(db *gorm.DB, logger *utility.Logger, fileIDs []string, channelID, messageID string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	updates := map[string]interface{}{}
	if channelID != "" {
		updates["channel_id"] = channelID
	}
	if messageID != "" {
		updates["message_id"] = messageID
	}

	if len(updates) == 0 {
		return nil
	}

	err := db.Model(&models.File{}).
		Where("id IN ?", fileIDs).
		Updates(updates).Error

	if err != nil {
		logger.Error("Failed to update file metadata",
			"file_ids", fileIDs,
			"channel_id", channelID,
			"message_id", messageID,
			"error", err)
		return fmt.Errorf("failed to update file metadata: %w", err)
	}

	return nil
}
