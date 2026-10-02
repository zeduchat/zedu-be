package agents

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

const (
	maxToolArgumentLength = 255
	maxSearchResults      = 100
	maxExportRows         = 1000
)

type ToolExecutionContext struct {
	DB        *gorm.DB
	AgentID   string
	OrgID     string
	UserID    string
	ChannelID string
	ThreadID  string
}

type AgentTool interface {
	Name() string
	Definition() models.Tool
	Execute(ToolExecutionContext, map[string]any) (any, error)
}

type ToolRegistry struct {
	tools map[string]AgentTool
}

func NewToolRegistry() *ToolRegistry {
	registry := &ToolRegistry{
		tools: make(map[string]AgentTool),
	}

	registry.Register(&capitalizeTextTool{})
	registry.Register(&searchOrganisationMembersTool{})
	registry.Register(&listChannelMembersTool{})
	registry.Register(&addUserToChannelTool{})
	registry.Register(&removeUserFromChannelTool{})
	registry.Register(&moveUserBetweenChannelsTool{})
	registry.Register(&exportChannelMembersTool{})

	return registry
}

func (r *ToolRegistry) Register(tool AgentTool) {
	if tool == nil {
		return
	}

	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Definitions() []models.Tool {
	if r == nil {
		return []models.Tool{}
	}

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	definitions := make([]models.Tool, 0, len(names))
	for _, name := range names {
		definitions = append(definitions, r.tools[name].Definition())
	}

	return definitions
}

func (r *ToolRegistry) Execute(ctx ToolExecutionContext, name string, arguments map[string]any) (any, error) {
	if r == nil {
		return nil, errors.New("tool registry is not initialized")
	}

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	if arguments == nil {
		arguments = map[string]any{}
	}

	return tool.Execute(ctx, arguments)
}

func ToolDefinitions() []models.Tool {
	return NewToolRegistry().Definitions()
}

func requirePermission(ctx ToolExecutionContext, permission string) error {
	if ctx.DB == nil {
		return errors.New("database is not initialized")
	}
	if strings.TrimSpace(ctx.OrgID) == "" {
		return errors.New("organisation context is required")
	}
	if strings.TrimSpace(ctx.AgentID) == "" {
		return errors.New("agent context is required")
	}
	if strings.TrimSpace(ctx.UserID) == "" {
		return errors.New("invoking user context is required")
	}

	var integration models.OrganisationIntegrations
	if err := ctx.DB.Where(
		"org_id = ? AND integration_id = ? AND is_archived = ? AND is_active = ?",
		ctx.OrgID,
		ctx.AgentID,
		false,
		true,
	).First(&integration).Error; err != nil {
		return errors.New("agent is not active in the organisation")
	}

	var membership models.OrgUserManagement
	if err := ctx.DB.Where(
		"organisation_id = ? AND user_id = ? AND is_deactivated = ?",
		ctx.OrgID,
		ctx.UserID,
		false,
	).First(&membership).Error; err != nil {
		return errors.New("invoking user is not an active organisation member")
	}

	roleInfo, err := membership.GetUserRoleInOrganisation(ctx.DB, ctx.UserID, ctx.OrgID)
	if err != nil {
		return fmt.Errorf("failed to resolve organisation permissions: %w", err)
	}

	for _, granted := range roleInfo.Permissions {
		if granted == permission {
			return nil
		}
	}

	return fmt.Errorf("permission denied: %s", permission)
}

func stringArgument(arguments map[string]any, key string, required bool) (string, error) {
	value, exists := arguments[key]
	if !exists || value == nil {
		if required {
			return "", fmt.Errorf("%s is required", key)
		}
		return "", nil
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}

	text = strings.TrimSpace(text)
	if required && text == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	if len(text) > maxToolArgumentLength {
		return "", fmt.Errorf("%s is too long", key)
	}

	return text, nil
}

func resolveOrganisationChannel(db *gorm.DB, orgID, channelName string) (models.Channels, error) {
	if db == nil {
		return models.Channels{}, errors.New("database is not initialized")
	}

	var channel models.Channels
	err := db.Where(
		"organisation_id = ? AND archived = ? AND LOWER(name) = LOWER(?)",
		orgID,
		false,
		channelName,
	).First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Channels{}, fmt.Errorf("channel not found: %s", channelName)
		}
		return models.Channels{}, fmt.Errorf("failed to resolve channel: %w", err)
	}

	return channel, nil
}

func resolveOrganisationUser(db *gorm.DB, orgID, identifier string) (models.UserInOrgResponse, error) {
	if db == nil {
		return models.UserInOrgResponse{}, errors.New("database is not initialized")
	}

	var user models.UserInOrgResponse
	err := db.Table("users").
		Select("users.id, users.email, COALESCE(profiles.user_name, '') AS username, COALESCE(profiles.phone, '') AS phone_number, COALESCE(profiles.avatar_url, '') AS avatar_url, users.name, COALESCE(org_roles.name, '') AS role, org_user_managements.status, org_user_managements.is_deactivated, users.created_at, 'user' AS entity_type, COALESCE(profiles.online, false) AS online").
		Joins("JOIN org_user_managements ON org_user_managements.user_id = users.id").
		Joins("LEFT JOIN org_roles ON org_roles.id = org_user_managements.role_id").
		Joins("LEFT JOIN profiles ON profiles.userid = users.id AND (profiles.organisation_id = ? OR profiles.organisation_id IS NULL)", orgID).
		Where("org_user_managements.organisation_id = ? AND org_user_managements.is_deactivated = ?", orgID, false).
		Where("(LOWER(users.email) = LOWER(?) OR LOWER(profiles.user_name) = LOWER(?) OR LOWER(users.name) = LOWER(?))", identifier, identifier, identifier).
		Order("users.name").
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.UserInOrgResponse{}, fmt.Errorf("user not found in organisation: %s", identifier)
		}
		return models.UserInOrgResponse{}, fmt.Errorf("failed to resolve organisation user: %w", err)
	}

	return user, nil
}

func channelMembers(db *gorm.DB, orgID, channelID string) ([]models.UserInOrgResponse, error) {
	var members []models.UserInOrgResponse
	err := db.Table("users").
		Select("users.id, users.email, COALESCE(profiles.user_name, '') AS username, COALESCE(profiles.phone, '') AS phone_number, COALESCE(profiles.avatar_url, '') AS avatar_url, users.name, COALESCE(org_roles.name, '') AS role, org_user_managements.status, org_user_managements.is_deactivated, users.created_at, 'user' AS entity_type, COALESCE(profiles.online, false) AS online").
		Joins("JOIN user_channels ON user_channels.user_id = users.id").
		Joins("JOIN org_user_managements ON org_user_managements.user_id = users.id AND org_user_managements.organisation_id = ?", orgID).
		Joins("LEFT JOIN org_roles ON org_roles.id = org_user_managements.role_id").
		Joins("LEFT JOIN profiles ON profiles.userid = users.id AND (profiles.organisation_id = ? OR profiles.organisation_id IS NULL)", orgID).
		Where("user_channels.channels_id = ?", channelID).
		Order("users.name").
		Find(&members).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list channel members: %w", err)
	}

	return members, nil
}

type capitalizeTextTool struct{}

func (t *capitalizeTextTool) Name() string {
	return "capitalize_text"
}

func (t *capitalizeTextTool) Definition() models.Tool {
	return toolDefinition(
		"capitalize_text",
		"Capitalizes all characters in the provided text to uppercase",
		[]string{"text"},
		map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "The text to capitalize",
			},
		},
	)
}

func (t *capitalizeTextTool) Execute(_ ToolExecutionContext, arguments map[string]any) (any, error) {
	text, err := stringArgument(arguments, "text", true)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"original":    text,
		"capitalized": strings.ToUpper(text),
	}, nil
}

type searchOrganisationMembersTool struct{}

func (t *searchOrganisationMembersTool) Name() string {
	return "search_organisation_members"
}

func (t *searchOrganisationMembersTool) Definition() models.Tool {
	return toolDefinition(
		"search_organisation_members",
		"Searches active and inactive members of the organisation by name, username, email, or phone.",
		[]string{"query"},
		map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Name, username, email, or phone number to search for.",
			},
		},
	)
}

func (t *searchOrganisationMembersTool) Execute(ctx ToolExecutionContext, arguments map[string]any) (any, error) {
	if err := requirePermission(ctx, models.PermViewChannels); err != nil {
		return nil, err
	}

	query, err := stringArgument(arguments, "query", true)
	if err != nil {
		return nil, err
	}

	var organisation models.Organisation
	if err := ctx.DB.Where("id = ?", ctx.OrgID).First(&organisation).Error; err != nil {
		return nil, errors.New("organisation not found")
	}

	var orgUserManagement models.OrgUserManagement
	members, err := orgUserManagement.SearchUsersInOrganisation(ctx.DB, ctx.OrgID, query)
	if err != nil {
		return nil, err
	}
	if len(members) > maxSearchResults {
		members = members[:maxSearchResults]
	}

	return map[string]any{
		"count":   len(members),
		"members": members,
	}, nil
}

type listChannelMembersTool struct{}

func (t *listChannelMembersTool) Name() string {
	return "list_channel_members"
}

func (t *listChannelMembersTool) Definition() models.Tool {
	return toolDefinition(
		"list_channel_members",
		"Lists the members of an organisation channel.",
		[]string{"channel_name"},
		map[string]any{
			"channel_name": map[string]any{
				"type":        "string",
				"description": "Name of the channel to inspect.",
			},
		},
	)
}

func (t *listChannelMembersTool) Execute(ctx ToolExecutionContext, arguments map[string]any) (any, error) {
	if err := requirePermission(ctx, models.PermViewChannels); err != nil {
		return nil, err
	}

	channelName, err := stringArgument(arguments, "channel_name", true)
	if err != nil {
		return nil, err
	}

	channel, err := resolveOrganisationChannel(ctx.DB, ctx.OrgID, channelName)
	if err != nil {
		return nil, err
	}

	members, err := channelMembers(ctx.DB, ctx.OrgID, channel.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"channel": channel.Name,
		"count":   len(members),
		"members": members,
	}, nil
}

type addUserToChannelTool struct{}

func (t *addUserToChannelTool) Name() string {
	return "add_user_to_channel"
}

func (t *addUserToChannelTool) Definition() models.Tool {
	return toolDefinition(
		"add_user_to_channel",
		"Adds an organisation member to a channel.",
		[]string{"channel_name", "username"},
		map[string]any{
			"channel_name": map[string]any{
				"type":        "string",
				"description": "Name of the target channel.",
			},
			"username": map[string]any{
				"type":        "string",
				"description": "Organisation username or email of the member.",
			},
		},
	)
}

func (t *addUserToChannelTool) Execute(ctx ToolExecutionContext, arguments map[string]any) (any, error) {
	if err := requirePermission(ctx, models.PermManageChannels); err != nil {
		return nil, err
	}

	channelName, err := stringArgument(arguments, "channel_name", true)
	if err != nil {
		return nil, err
	}
	username, err := stringArgument(arguments, "username", true)
	if err != nil {
		return nil, err
	}

	channel, err := resolveOrganisationChannel(ctx.DB, ctx.OrgID, channelName)
	if err != nil {
		return nil, err
	}
	user, err := resolveOrganisationUser(ctx.DB, ctx.OrgID, username)
	if err != nil {
		return nil, err
	}

	if _, err := channel.AddUserToChannel(ctx.DB, models.JoinChannelsRequest{
		ChannelsID: channel.ID,
		UserID:     user.ID,
		Username:   user.UserName,
	}); err != nil {
		return nil, err
	}

	return map[string]any{
		"status":   "added",
		"channel":  channel.Name,
		"user_id":  user.ID,
		"username": user.UserName,
	}, nil
}

type removeUserFromChannelTool struct{}

func (t *removeUserFromChannelTool) Name() string {
	return "remove_user_from_channel"
}

func (t *removeUserFromChannelTool) Definition() models.Tool {
	return toolDefinition(
		"remove_user_from_channel",
		"Removes an organisation member from a channel.",
		[]string{"channel_name", "username"},
		map[string]any{
			"channel_name": map[string]any{
				"type":        "string",
				"description": "Name of the target channel.",
			},
			"username": map[string]any{
				"type":        "string",
				"description": "Organisation username or email of the member.",
			},
		},
	)
}

func (t *removeUserFromChannelTool) Execute(ctx ToolExecutionContext, arguments map[string]any) (any, error) {
	if err := requirePermission(ctx, models.PermManageChannels); err != nil {
		return nil, err
	}

	channelName, err := stringArgument(arguments, "channel_name", true)
	if err != nil {
		return nil, err
	}
	username, err := stringArgument(arguments, "username", true)
	if err != nil {
		return nil, err
	}

	channel, err := resolveOrganisationChannel(ctx.DB, ctx.OrgID, channelName)
	if err != nil {
		return nil, err
	}
	user, err := resolveOrganisationUser(ctx.DB, ctx.OrgID, username)
	if err != nil {
		return nil, err
	}

	if user.ID == channel.OwnerId {
		return nil, errors.New("cannot remove the channel owner")
	}

	if err := channel.RemoveUserFromChannels(ctx.DB, channel.ID, user.ID); err != nil {
		return nil, err
	}

	return map[string]any{
		"status":   "removed",
		"channel":  channel.Name,
		"user_id":  user.ID,
		"username": user.UserName,
	}, nil
}

type moveUserBetweenChannelsTool struct{}

func (t *moveUserBetweenChannelsTool) Name() string {
	return "move_user_between_channels"
}

func (t *moveUserBetweenChannelsTool) Definition() models.Tool {
	return toolDefinition(
		"move_user_between_channels",
		"Moves an organisation member from one channel to another.",
		[]string{"username", "from_channel", "to_channel"},
		map[string]any{
			"username": map[string]any{
				"type":        "string",
				"description": "Organisation username or email of the member.",
			},
			"from_channel": map[string]any{
				"type":        "string",
				"description": "Source channel name.",
			},
			"to_channel": map[string]any{
				"type":        "string",
				"description": "Destination channel name.",
			},
		},
	)
}

func (t *moveUserBetweenChannelsTool) Execute(ctx ToolExecutionContext, arguments map[string]any) (any, error) {
	if err := requirePermission(ctx, models.PermManageChannels); err != nil {
		return nil, err
	}

	username, err := stringArgument(arguments, "username", true)
	if err != nil {
		return nil, err
	}
	fromName, err := stringArgument(arguments, "from_channel", true)
	if err != nil {
		return nil, err
	}
	toName, err := stringArgument(arguments, "to_channel", true)
	if err != nil {
		return nil, err
	}

	fromChannel, err := resolveOrganisationChannel(ctx.DB, ctx.OrgID, fromName)
	if err != nil {
		return nil, err
	}
	toChannel, err := resolveOrganisationChannel(ctx.DB, ctx.OrgID, toName)
	if err != nil {
		return nil, err
	}
	if fromChannel.ID == toChannel.ID {
		return nil, errors.New("source and destination channels must differ")
	}

	user, err := resolveOrganisationUser(ctx.DB, ctx.OrgID, username)
	if err != nil {
		return nil, err
	}
	if user.ID == fromChannel.OwnerId {
		return nil, errors.New("cannot move the channel owner")
	}

	var membership models.UserChannels
	if err := ctx.DB.Where("channels_id = ? AND user_id = ?", fromChannel.ID, user.ID).First(&membership).Error; err != nil {
		return nil, errors.New("user is not a member of the source channel")
	}

	tx := ctx.DB.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to start channel move: %w", tx.Error)
	}

	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	if err := fromChannel.RemoveUserFromChannels(tx, fromChannel.ID, user.ID); err != nil {
		return nil, err
	}
	if _, err := toChannel.AddUserToChannel(tx, models.JoinChannelsRequest{
		ChannelsID: toChannel.ID,
		UserID:     user.ID,
		Username:   user.UserName,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit channel move: %w", err)
	}
	committed = true

	return map[string]any{
		"status":       "moved",
		"user_id":      user.ID,
		"username":     user.UserName,
		"from_channel": fromChannel.Name,
		"to_channel":   toChannel.Name,
	}, nil
}

type exportChannelMembersTool struct{}

func (t *exportChannelMembersTool) Name() string {
	return "export_channel_members"
}

func (t *exportChannelMembersTool) Definition() models.Tool {
	return toolDefinition(
		"export_channel_members",
		"Exports channel members as CSV text.",
		[]string{"channel_name"},
		map[string]any{
			"channel_name": map[string]any{
				"type":        "string",
				"description": "Name of the channel to export.",
			},
		},
	)
}

func (t *exportChannelMembersTool) Execute(ctx ToolExecutionContext, arguments map[string]any) (any, error) {
	if err := requirePermission(ctx, models.PermViewChannels); err != nil {
		return nil, err
	}

	channelName, err := stringArgument(arguments, "channel_name", true)
	if err != nil {
		return nil, err
	}

	channel, err := resolveOrganisationChannel(ctx.DB, ctx.OrgID, channelName)
	if err != nil {
		return nil, err
	}

	members, err := channelMembers(ctx.DB, ctx.OrgID, channel.ID)
	if err != nil {
		return nil, err
	}

	truncated := false
	if len(members) > maxExportRows {
		members = members[:maxExportRows]
		truncated = true
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"id", "name", "username", "email", "role", "status", "online"}); err != nil {
		return nil, fmt.Errorf("failed to build CSV export: %w", err)
	}

	for _, member := range members {
		if err := writer.Write([]string{
			member.ID,
			member.Name,
			member.UserName,
			member.Email,
			member.Role,
			member.Status,
			fmt.Sprintf("%t", member.Online),
		}); err != nil {
			return nil, fmt.Errorf("failed to build CSV export: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to finalize CSV export: %w", err)
	}

	return map[string]any{
		"channel":   channel.Name,
		"count":     len(members),
		"truncated": truncated,
		"format":    "csv",
		"csv":       buffer.String(),
	}, nil
}

func toolDefinition(name, description string, required []string, properties map[string]any) models.Tool {
	return models.Tool{
		Type: "function",
		Function: models.ToolFunction{
			Name:        name,
			Description: description,
			Parameters: models.ToolFunctionParameter{
				Type:       "object",
				Properties: properties,
				Required:   required,
			},
		},
	}
}
