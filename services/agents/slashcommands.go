package agents

import (
	"fmt"
	"strings"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func AddAgentsSlashCommand(db *gorm.DB, ids map[string]string, req models.AddSlashCommandRequest) (models.SlashCommand, error) {

	orgID := ids["org_id"]
	integrationID := ids["agent_id"]

	slashCommand := models.SlashCommand{
		ID:            utility.GenerateUUID(),
		OrgID:         &orgID,
		IntegrationID: &integrationID,
		Command:       req.Command,
		ProcessingURL: req.ProcessingURL,
		Description:   req.Description,
	}

	response, err := slashCommand.CreateSlashCommand(db)
	if err != nil {
		return response, err
	}

	return response, nil
}

func GetIntegrationSlashCommands(db *gorm.DB, ids map[string]string) ([]models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.GetIntegrationSlashCommands(db, ids)
	if err != nil {
		return response, err
	}
	return response, nil
}

func GetAllOrgSlashCommands(db *gorm.DB, orgID string) ([]models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.GetAllOrgSlashCommands(db, orgID)
	if err != nil {
		return response, err
	}
	return response, nil
}

func UpdateAgentSlashCommand(db *gorm.DB, ids map[string]string, req models.UpdateSlashCommandRequest) (models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.UpdateSlashCommand(db, ids, req)
	if err != nil {
		return response, err
	}
	return response, err
}

func DeleteAgentSlashCommand(db *gorm.DB, ids map[string]string) error {
	var (
		slashCommand models.SlashCommand
	)

	err := slashCommand.DeleteSlashCommand(db, ids)
	if err != nil {
		return err
	}
	return nil
}

func GetDefaultSlashCommands(db *gorm.DB) ([]models.SlashCommand, error) {
	var (
		slashCommand models.SlashCommand
	)

	response, err := slashCommand.GetDefaultSlashCommands(db)
	if err != nil {
		return response, err
	}
	return response, nil
}

func ProcessSlashCommand(db *gorm.DB, req models.ProcessSlashCommandRequest) (map[string]interface{}, error) {
	// Parse the command to extract command type and arguments
	commandType, args, err := parseSlashCommand(req.Command)
	if err != nil {
		return nil, err
	}

	// Route to appropriate handler
	switch commandType {
	case "add-to-channel":
		return handleAddToChannel(db, args)
	case "remove-from-channel":
		return handleRemoveFromChannel(db, args)
	case "banish-from-channel":
		return handleBanishFromChannel(db, args)
	case "restore-channels":
		return handleRestoreChannels(db, args)
	case "export-members":
		return handleExportMembers(db, args)
	case "promote":
		return handlePromote(db, args)
	case "demote":
		return handleDemote(db, args)
	case "add-to-all-org-channels":
		return handleAddToAllOrgChannels(db, req, args)
	default:
		return nil, fmt.Errorf("unknown slash command: %s", commandType)
	}
}

// parseSlashCommand parses a slash command string and returns the command type and arguments
func parseSlashCommand(command string) (string, map[string]string, error) {
	// Remove leading slash and split by spaces
	parts := strings.Fields(strings.TrimPrefix(command, "/"))
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	commandType := parts[0]
	args := make(map[string]string)

	// Parse arguments - channels start with #, users start with @
	var channels []string
	var users []string

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "#") {
			channels = append(channels, strings.TrimPrefix(part, "#"))
		} else if strings.HasPrefix(part, "@") {
			users = append(users, strings.TrimPrefix(part, "@"))
		}
	}

	if len(channels) > 0 {
		args["channels"] = strings.Join(channels, ",")
	}
	if len(users) > 0 {
		args["users"] = strings.Join(users, ",")
	}

	// For commands with specific structure
	switch commandType {
	case "add-to-channel", "remove-from-channel":
		// Format: /command #channel-name @user1 @user2 ...
		if len(channels) == 0 {
			return "", nil, fmt.Errorf("no channel specified")
		}
		args["target_channel"] = channels[0]
		// Remaining users
		if len(users) > 0 {
			args["users"] = strings.Join(users, ",")
		}
	case "promote", "demote", "banish-from-channel":
		// Format: /command #target-channel @user1 @user2 ... #channel1 #channel2
		if len(channels) == 0 {
			return "", nil, fmt.Errorf("no target channel specified")
		}
		args["target_channel"] = channels[0]
		// Source channels (remaining channels)
		if len(channels) > 1 {
			args["source_channels"] = strings.Join(channels[1:], ",")
		}
		// Users
		if len(users) > 0 {
			args["users"] = strings.Join(users, ",")
		}
	case "restore-channels":
		// Format: /restore-channels @user1 @user2 ...
		if len(users) == 0 {
			return "", nil, fmt.Errorf("no users specified")
		}
		args["users"] = strings.Join(users, ",")
	case "add-to-all-org-channels":
		// Format: /add-to-all-org-channels @username
		if len(users) == 0 {
			return "", nil, fmt.Errorf("username is required")
		}
		args["users"] = strings.Join(users, ",")
	}

	return commandType, args, nil
}

// getChannelByName retrieves a channel by its name within an organization
func getChannelByName(db *gorm.DB, channelName, orgID string) (*models.Channels, error) {
	var channel models.Channels
	err := db.Where("name = ? AND organisation_id = ?", channelName, orgID).First(&channel).Error
	if err != nil {
		return nil, fmt.Errorf("channel not found: %s", channelName)
	}
	return &channel, nil
}

// getUserByUsername retrieves a user by their username
func getUserByUsername(db *gorm.DB, username string) (*models.User, error) {
	var profile models.Profile
	err := db.Where("user_name = ?", username).First(&profile).Error
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	var user models.User
	err = db.Where("id = ?", profile.Userid).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("user not found for profile: %s", username)
	}
	return &user, nil
}

// handleAddToChannel adds users to a channel
func handleAddToChannel(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	channelName := args["target_channel"]
	usersStr := args["users"]

	if channelName == "" {
		return nil, fmt.Errorf("channel name is required")
	}
	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}

	// Get channel
	var channel models.Channels
	exists := postgresql.CheckExists(db, &channel, "name = ?", channelName)
	if !exists {
		return nil, fmt.Errorf("channel not found: %s", channelName)
	}

	// Get channel details
	err := db.Where("name = ?", channelName).First(&channel).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %v", err)
	}

	users := strings.Split(usersStr, ",")
	addedUsers := []string{}
	failedUsers := []map[string]string{}

	for _, username := range users {
		user, err := getUserByUsername(db, username)
		if err != nil {
			failedUsers = append(failedUsers, map[string]string{
				"username": username,
				"error":    err.Error(),
			})
			continue
		}

		// Check if user is already in channel
		var userChannel models.UserChannels
		exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", channel.ID, user.ID)
		if exists {
			failedUsers = append(failedUsers, map[string]string{
				"username": username,
				"error":    "user already in channel",
			})
			continue
		}

		// Add user to channel
		req := models.JoinChannelsRequest{
			ChannelsID: channel.ID,
			UserID:     user.ID,
			Username:   username,
		}

		_, err = channel.AddUserToChannel(db, req)
		if err != nil {
			failedUsers = append(failedUsers, map[string]string{
				"username": username,
				"error":    err.Error(),
			})
			continue
		}

		addedUsers = append(addedUsers, username)
	}

	return map[string]interface{}{
		"status":       "completed",
		"message":      fmt.Sprintf("Add to channel operation completed"),
		"channel":      channelName,
		"added_users":  addedUsers,
		"failed_users": failedUsers,
		"added_count":  len(addedUsers),
		"failed_count": len(failedUsers),
	}, nil
}

// handleRemoveFromChannel removes users from a channel
func handleRemoveFromChannel(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	channelName := args["target_channel"]
	usersStr := args["users"]

	if channelName == "" {
		return nil, fmt.Errorf("channel name is required")
	}
	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}

	// Get channel
	var channel models.Channels
	err := db.Where("name = ?", channelName).First(&channel).Error
	if err != nil {
		return nil, fmt.Errorf("channel not found: %s", channelName)
	}

	users := strings.Split(usersStr, ",")
	removedUsers := []string{}
	failedUsers := []map[string]string{}

	for _, username := range users {
		user, err := getUserByUsername(db, username)
		if err != nil {
			failedUsers = append(failedUsers, map[string]string{
				"username": username,
				"error":    err.Error(),
			})
			continue
		}

		// Remove user from channel
		err = channel.RemoveUserFromChannels(db, channel.ID, user.ID)
		if err != nil {
			failedUsers = append(failedUsers, map[string]string{
				"username": username,
				"error":    err.Error(),
			})
			continue
		}

		removedUsers = append(removedUsers, username)
	}

	return map[string]interface{}{
		"status":        "completed",
		"message":       fmt.Sprintf("Remove from channel operation completed"),
		"channel":       channelName,
		"removed_users": removedUsers,
		"failed_users":  failedUsers,
		"removed_count": len(removedUsers),
		"failed_count":  len(failedUsers),
	}, nil
}

// handleBanishFromChannel removes users from specified channels and adds them to a target channel
func handleBanishFromChannel(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	targetChannelName := args["target_channel"]
	sourceChannelsStr := args["source_channels"]
	usersStr := args["users"]

	if targetChannelName == "" {
		return nil, fmt.Errorf("target channel name is required")
	}
	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}

	// Get target channel
	var targetChannel models.Channels
	err := db.Where("name = ?", targetChannelName).First(&targetChannel).Error
	if err != nil {
		return nil, fmt.Errorf("target channel not found: %s", targetChannelName)
	}

	users := strings.Split(usersStr, ",")
	sourceChannels := []string{}
	if sourceChannelsStr != "" {
		sourceChannels = strings.Split(sourceChannelsStr, ",")
	}

	results := []map[string]interface{}{}

	for _, username := range users {
		user, err := getUserByUsername(db, username)
		if err != nil {
			results = append(results, map[string]interface{}{
				"username": username,
				"status":   "failed",
				"error":    err.Error(),
			})
			continue
		}

		// Remove from source channels and track history
		removedFrom := []string{}
		removedFromIDs := []string{}

		for _, channelName := range sourceChannels {
			var sourceChannel models.Channels
			err := db.Where("name = ?", channelName).First(&sourceChannel).Error
			if err != nil {
				continue
			}

			var userChannel models.UserChannels
			exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", sourceChannel.ID, user.ID)
			if exists {
				err = sourceChannel.RemoveUserFromChannels(db, sourceChannel.ID, user.ID)
				if err == nil {
					removedFrom = append(removedFrom, channelName)
					removedFromIDs = append(removedFromIDs, sourceChannel.ID)
				}
			}
		}

		// Save channel history (one record with all channel IDs)
		if len(removedFromIDs) > 0 {
			history := models.UserChannelHistory{
				ID:                  utility.GenerateUUID(),
				UserID:              user.ID,
				OrganisationID:      targetChannel.OrganisationID,
				ChannelIDs:          removedFromIDs,
				BanishedToChannelID: &targetChannel.ID,
				Action:              "banished",
			}
			_ = history.SaveChannelHistory(db)
		}

		// Add to target channel
		var userChannel models.UserChannels
		exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", targetChannel.ID, user.ID)

		if !exists {
			req := models.JoinChannelsRequest{
				ChannelsID: targetChannel.ID,
				UserID:     user.ID,
				Username:   username,
			}
			_, _ = targetChannel.AddUserToChannel(db, req)
		}

		results = append(results, map[string]interface{}{
			"username":     username,
			"status":       "success",
			"removed_from": removedFrom,
			"added_to":     targetChannelName,
		})
	}

	return map[string]interface{}{
		"status":         "completed",
		"message":        "Banish from channel operation completed",
		"target_channel": targetChannelName,
		"results":        results,
	}, nil
}

// handleRestoreChannels restores channel access for users by re-adding them to previously banished channels
func handleRestoreChannels(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	usersStr := args["users"]

	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}

	users := strings.Split(usersStr, ",")
	results := []map[string]interface{}{}

	for _, username := range users {
		user, err := getUserByUsername(db, username)
		if err != nil {
			results = append(results, map[string]interface{}{
				"username": username,
				"status":   "failed",
				"error":    err.Error(),
			})
			continue
		}

		// Get all unique channel IDs from banished history
		var history models.UserChannelHistory
		banishedChannelIDs, err := history.GetBanishedChannelIDs(db, user.ID)
		if err != nil {
			results = append(results, map[string]interface{}{
				"username": username,
				"status":   "failed",
				"error":    fmt.Sprintf("failed to get banished channels: %v", err),
			})
			continue
		}

		if len(banishedChannelIDs) == 0 {
			results = append(results, map[string]interface{}{
				"username": username,
				"status":   "success",
				"message":  "No banished channels found to restore",
			})
			continue
		}

		// Restore channels
		restoredChannels := []string{}
		restoredChannelIDs := []string{}

		for _, channelID := range banishedChannelIDs {
			var channel models.Channels
			err := db.Where("id = ?", channelID).First(&channel).Error
			if err != nil {
				continue
			}

			// Check if user is already in channel
			var userChannel models.UserChannels
			exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", channel.ID, user.ID)
			if exists {
				continue
			}

			// Add user back to channel
			req := models.JoinChannelsRequest{
				ChannelsID: channel.ID,
				UserID:     user.ID,
				Username:   username,
			}

			_, err = channel.AddUserToChannel(db, req)
			if err != nil {
				continue
			}

			restoredChannels = append(restoredChannels, channel.Name)
			restoredChannelIDs = append(restoredChannelIDs, channel.ID)
		}

		// Mark channels as restored in history
		if len(restoredChannelIDs) > 0 {
			err = history.MarkChannelsAsRestored(db, user.ID, restoredChannelIDs)
			if err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Warning: failed to mark channels as restored for user %s: %v\n", username, err)
			}
		}

		results = append(results, map[string]interface{}{
			"username":          username,
			"status":            "success",
			"restored_channels": restoredChannels,
			"restored_count":    len(restoredChannels),
		})
	}

	return map[string]interface{}{
		"status":  "completed",
		"message": "Restore channels operation completed",
		"results": results,
	}, nil
}

// handleExportMembers exports channel members as CSV
// Exports: name, email, and role for all members in a channel
func handleExportMembers(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	channelName := args["channel"]

	if channelName == "" {
		return nil, fmt.Errorf("channel name is required")
	}

	// Get the channel by name
	var channel models.Channels
	err := db.Where("name = ?", channelName).First(&channel).Error
	if err != nil {
		return nil, fmt.Errorf("channel not found: %s", channelName)
	}

	// Get all users in the channel
	var users []models.User
	err = db.Table("users").
		Select("users.id, users.name, users.email, users.role").
		Joins("JOIN user_channels ON user_channels.user_id = users.id").
		Where("user_channels.channels_id = ?", channel.ID).
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel members: %v", err)
	}

	if len(users) == 0 {
		return map[string]interface{}{
			"status":  "completed",
			"message": "No members found in channel",
			"data":    []string{},
		}, nil
	}

	// Build CSV header
	csvLines := []string{"Name,Email,Role"}

	// Build CSV data rows
	for _, user := range users {
		role := getRoleString(user.Role)
		csvLines = append(csvLines, fmt.Sprintf("%s,%s,%s", user.Name, user.Email, role))
	}

	// Combine all lines into a single CSV string
	csvContent := strings.Join(csvLines, "\n")

	return map[string]interface{}{
		"status":  "completed",
		"message": fmt.Sprintf("Exported %d members from channel '%s'", len(users), channelName),
		"data": map[string]interface{}{
			"channel_name":   channelName,
			"member_count":   len(users),
			"csv_content":    csvContent,
			"content_type":   "text/csv",
			"file_suggested": fmt.Sprintf("%s_members.csv", channelName),
		},
	}, nil
}

// getRoleString converts role integer to string representation
func getRoleString(role int) string {
	switch role {
	case 1:
		return "admin"
	case 2:
		return "super-admin"
	default:
		return "user"
	}
}

// handlePromote promotes users from specified channels to a target channel
func handlePromote(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	targetChannelName := args["target_channel"]
	sourceChannelsStr := args["source_channels"]
	usersStr := args["users"]

	if targetChannelName == "" {
		return nil, fmt.Errorf("target channel name is required")
	}
	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}
	if sourceChannelsStr == "" {
		return nil, fmt.Errorf("at least one source channel is required")
	}

	// Get target channel
	var targetChannel models.Channels
	err := db.Where("name = ?", targetChannelName).First(&targetChannel).Error
	if err != nil {
		return nil, fmt.Errorf("target channel not found: %s", targetChannelName)
	}

	users := strings.Split(usersStr, ",")
	sourceChannels := strings.Split(sourceChannelsStr, ",")
	results := []map[string]interface{}{}

	for _, username := range users {
		user, err := getUserByUsername(db, username)
		if err != nil {
			results = append(results, map[string]interface{}{
				"username": username,
				"status":   "failed",
				"error":    err.Error(),
			})
			continue
		}

		// Check if user is in source channels and remove them
		removedFrom := []string{}
		for _, channelName := range sourceChannels {
			var sourceChannel models.Channels
			err := db.Where("name = ?", channelName).First(&sourceChannel).Error
			if err != nil {
				continue
			}

			var userChannel models.UserChannels
			exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", sourceChannel.ID, user.ID)
			if exists {
				err = sourceChannel.RemoveUserFromChannels(db, sourceChannel.ID, user.ID)
				if err == nil {
					removedFrom = append(removedFrom, channelName)
				}
			}
		}

		// Add to target channel
		var userChannel models.UserChannels
		exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", targetChannel.ID, user.ID)

		if !exists {
			req := models.JoinChannelsRequest{
				ChannelsID: targetChannel.ID,
				UserID:     user.ID,
				Username:   username,
			}
			_, err = targetChannel.AddUserToChannel(db, req)
			if err != nil {
				results = append(results, map[string]interface{}{
					"username":     username,
					"status":       "partial_success",
					"removed_from": removedFrom,
					"error":        fmt.Sprintf("failed to add to target channel: %v", err),
				})
				continue
			}
		}

		results = append(results, map[string]interface{}{
			"username":     username,
			"status":       "success",
			"removed_from": removedFrom,
			"promoted_to":  targetChannelName,
		})
	}

	return map[string]interface{}{
		"status":         "completed",
		"message":        "Promote operation completed",
		"target_channel": targetChannelName,
		"results":        results,
	}, nil
}

// handleDemote demotes users from specified channels to a target channel
func handleDemote(db *gorm.DB, args map[string]string) (map[string]interface{}, error) {
	targetChannelName := args["target_channel"]
	sourceChannelsStr := args["source_channels"]
	usersStr := args["users"]

	if targetChannelName == "" {
		return nil, fmt.Errorf("target channel name is required")
	}
	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}
	if sourceChannelsStr == "" {
		return nil, fmt.Errorf("at least one source channel is required")
	}

	// Get target channel
	var targetChannel models.Channels
	err := db.Where("name = ?", targetChannelName).First(&targetChannel).Error
	if err != nil {
		return nil, fmt.Errorf("target channel not found: %s", targetChannelName)
	}

	users := strings.Split(usersStr, ",")
	sourceChannels := strings.Split(sourceChannelsStr, ",")
	results := []map[string]interface{}{}

	for _, username := range users {
		user, err := getUserByUsername(db, username)
		if err != nil {
			results = append(results, map[string]interface{}{
				"username": username,
				"status":   "failed",
				"error":    err.Error(),
			})
			continue
		}

		// Remove from source channels
		removedFrom := []string{}
		for _, channelName := range sourceChannels {
			var sourceChannel models.Channels
			err := db.Where("name = ?", channelName).First(&sourceChannel).Error
			if err != nil {
				continue
			}

			var userChannel models.UserChannels
			exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", sourceChannel.ID, user.ID)
			if exists {
				err = sourceChannel.RemoveUserFromChannels(db, sourceChannel.ID, user.ID)
				if err == nil {
					removedFrom = append(removedFrom, channelName)
				}
			}
		}

		// Add to target channel
		var userChannel models.UserChannels
		exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", targetChannel.ID, user.ID)

		if !exists {
			req := models.JoinChannelsRequest{
				ChannelsID: targetChannel.ID,
				UserID:     user.ID,
				Username:   username,
			}
			_, err = targetChannel.AddUserToChannel(db, req)
			if err != nil {
				results = append(results, map[string]interface{}{
					"username":     username,
					"status":       "partial_success",
					"removed_from": removedFrom,
					"error":        fmt.Sprintf("failed to add to target channel: %v", err),
				})
				continue
			}
		}

		results = append(results, map[string]interface{}{
			"username":     username,
			"status":       "success",
			"removed_from": removedFrom,
			"demoted_to":   targetChannelName,
		})
	}

	return map[string]interface{}{
		"status":         "completed",
		"message":        "Demote operation completed",
		"target_channel": targetChannelName,
		"results":        results,
	}, nil
}

// handleAddToAllOrgChannels adds a user to all organization channels
func handleAddToAllOrgChannels(db *gorm.DB, req models.ProcessSlashCommandRequest, args map[string]string) (map[string]interface{}, error) {
	usersStr := args["users"]

	if usersStr == "" {
		return nil, fmt.Errorf("at least one user is required")
	}

	users := strings.Split(usersStr, ",")
	if len(users) != 1 {
		return nil, fmt.Errorf("this command only accepts one username")
	}

	username := users[0]

	// Get the invoker's organization ID from context
	orgIDInterface, ok := req.Context["org_id"]
	if !ok || orgIDInterface == nil {
		return nil, fmt.Errorf("organization context is required")
	}

	orgID, ok := orgIDInterface.(string)
	if !ok || orgID == "" {
		return nil, fmt.Errorf("invalid organization context")
	}

	// Get the user to be added
	user, err := getUserByUsername(db, username)
	if err != nil {
		return nil, err
	}

	// Get all channels in the invoker's organization
	var channels []models.Channels
	err = db.Where("organisation_id = ?", orgID).Find(&channels).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization channels: %v", err)
	}

	if len(channels) == 0 {
		return map[string]interface{}{
			"status":  "completed",
			"message": "No channels found in organization",
			"results": []map[string]interface{}{},
		}, nil
	}

	addedChannels := []string{}
	skippedChannels := []map[string]string{}

	for _, channel := range channels {
		// Check if user is already in channel
		var userChannel models.UserChannels
		exists := postgresql.CheckExists(db, &userChannel, "channels_id = ? AND user_id = ?", channel.ID, user.ID)
		if exists {
			skippedChannels = append(skippedChannels, map[string]string{
				"channel": channel.Name,
				"reason":  "already a member",
			})
			continue
		}

		// Add user to channel
		joinReq := models.JoinChannelsRequest{
			ChannelsID: channel.ID,
			UserID:     user.ID,
			Username:   username,
		}

		_, err = channel.AddUserToChannel(db, joinReq)
		if err != nil {
			skippedChannels = append(skippedChannels, map[string]string{
				"channel": channel.Name,
				"reason":  err.Error(),
			})
			continue
		}

		addedChannels = append(addedChannels, channel.Name)
	}

	return map[string]interface{}{
		"status":          "completed",
		"message":         fmt.Sprintf("Added %s to all organization channels", username),
		"username":        username,
		"added_channels":  addedChannels,
		"added_count":     len(addedChannels),
		"skipped_channels": skippedChannels,
		"skipped_count":   len(skippedChannels),
		"organisation_id": orgID,
	}, nil
}
