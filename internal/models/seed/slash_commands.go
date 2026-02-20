package seed

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func SeedDefaultSlashCommands(logger *utility.Logger, db *gorm.DB) {
	defaultCommands := []models.SlashCommand{
		{
			ID:        utility.GenerateUUID(),
			Command:   "/add-to-channel #channel-name @user1 @user2 ... @user_n",
			IsDefault: true,
			Description: "Add one or more users to a channel",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/remove-from-channel #channel-name @user1 @user2 ... @user_n",
			IsDefault: true,
			Description: "Remove one or more users from a channel",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/banish-from-channel #channel-to-banish-to @user1 @user2 ... @user_n #channel1 #channel2",
			IsDefault: true,
			Description: "Banish users from specified channels to a specific channel",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/restore-channels @user1 @user2 ... @user_n",
			IsDefault: true,
			Description: "Restore channel access for one or more users",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/export-members",
			IsDefault: true,
			Description: "Export members list",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/promote #channel-to-promote-to @user1 @user2 ... @user_n #channel1 #channel2",
			IsDefault: true,
			Description: "Promote users from specified channels to a target channel",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/demote #channel-to-demote-to @user1 @user2 ... @user_n #channel1 #channel2",
			IsDefault: true,
			Description: "Demote users from specified channels to a target channel",
		},
		{
			ID:        utility.GenerateUUID(),
			Command:   "/add-to-all-org-channels @username",
			IsDefault: true,
			Description: "Add a user to all organization channels",
		},
	}

	for _, cmd := range defaultCommands {
		var existingCmd models.SlashCommand
		if err := db.Where("command = ? AND is_default = ?", cmd.Command, true).First(&existingCmd).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&cmd).Error; err != nil {
					logger.Error(fmt.Sprintf("Failed to seed default slash command: %s, error: %v", cmd.Command, err))
				} else {
					logger.Info(fmt.Sprintf("Successfully seeded default slash command: %s", cmd.Command))
				}
			} else {
				logger.Error(fmt.Sprintf("Error checking for default slash command %s: %v", cmd.Command, err))
			}
		}
	}
}
