package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type SlashCommand struct {
	ID            string     `gorm:"type:uuid;primary_key" json:"id"`
	OrgID         *string    `gorm:"type:uuid;" json:"org_id,omitempty"`
	IntegrationID *string    `gorm:"type:uuid;" json:"integration_id,omitempty"`
	Command       string     `gorm:"column:command; type:varchar(255);" json:"command"`
	ProcessingURL string     `gorm:"column:processing_url; type:varchar(255);" json:"processing_url"`
	CreatedAt     time.Time  `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	Description   string     `gorm:"column:description; type:varchar(255);" json:"description"`
	UpdatedAt     time.Time  `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at,omitempty"`
	IsDefault     bool       `gorm:"column:is_default; default:false;" json:"is_default"`
}

type AddSlashCommandRequest struct {
	Command       string `json:"command" validate:"required"`
	ProcessingURL string `json:"processing_url" validate:"required"`
	Description   string `json:"description"`
}

type UpdateSlashCommandRequest struct {
	Command       string `json:"command"`
	ProcessingURL string `json:"processing_url"`
	Description   string `json:"description"`
}

type ProcessSlashCommandRequest struct {
	Command  string                 `json:"command" validate:"required"`
	Context  map[string]interface{} `json:"context"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Slash command response types

type FailedUser struct {
	Username string `json:"username"`
	Error    string `json:"error"`
}

type SkippedChannel struct {
	Channel string `json:"channel"`
	Reason  string `json:"reason"`
}

type AddToChannelResponse struct {
	Status      string       `json:"status"`
	Message     string       `json:"message"`
	Channel     string       `json:"channel"`
	AddedUsers  []string     `json:"added_users"`
	FailedUsers []FailedUser `json:"failed_users"`
	AddedCount  int          `json:"added_count"`
	FailedCount int          `json:"failed_count"`
}

type RemoveFromChannelResponse struct {
	Status        string       `json:"status"`
	Message       string       `json:"message"`
	Channel       string       `json:"channel"`
	RemovedUsers  []string     `json:"removed_users"`
	FailedUsers   []FailedUser `json:"failed_users"`
	RemovedCount  int          `json:"removed_count"`
	FailedCount   int          `json:"failed_count"`
}

type BanishResult struct {
	Username     string   `json:"username"`
	Status       string   `json:"status"`
	RemovedFrom  []string `json:"removed_from,omitempty"`
	AddedTo      string   `json:"added_to,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type BanishFromChannelResponse struct {
	Status         string         `json:"status"`
	Message        string         `json:"message"`
	TargetChannel  string         `json:"target_channel"`
	Results        []BanishResult `json:"results"`
}

type RestoreResult struct {
	Username          string   `json:"username"`
	Status            string   `json:"status"`
	RestoredChannels  []string `json:"restored_channels,omitempty"`
	RestoredCount     int      `json:"restored_count,omitempty"`
	Message           string   `json:"message,omitempty"`
	Error             string   `json:"error,omitempty"`
}

type RestoreChannelsResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Results []RestoreResult `json:"results"`
}

type ExportMembersData struct {
	ChannelName   string `json:"channel_name"`
	MemberCount   int    `json:"member_count"`
	CSVContent    string `json:"csv_content"`
	ContentType   string `json:"content_type"`
	FileSuggested string `json:"file_suggested"`
}

type ExportMembersResponse struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    ExportMembersData   `json:"data"`
}

type PromoteDemoteResult struct {
	Username     string   `json:"username"`
	Status       string   `json:"status"`
	RemovedFrom  []string `json:"removed_from,omitempty"`
	PromotedTo   string   `json:"promoted_to,omitempty"`
	DemotedTo    string   `json:"demoted_to,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type PromoteResponse struct {
	Status         string             `json:"status"`
	Message        string             `json:"message"`
	TargetChannel  string             `json:"target_channel"`
	Results        []PromoteDemoteResult `json:"results"`
}

type DemoteResponse struct {
	Status         string             `json:"status"`
	Message        string             `json:"message"`
	TargetChannel  string             `json:"target_channel"`
	Results        []PromoteDemoteResult `json:"results"`
}

type AddToAllOrgChannelsResponse struct {
	Status          string            `json:"status"`
	Message         string            `json:"message"`
	Username        string            `json:"username"`
	AddedChannels   []string          `json:"added_channels"`
	AddedCount      int               `json:"added_count"`
	SkippedChannels []SkippedChannel  `json:"skipped_channels"`
	SkippedCount    int               `json:"skipped_count"`
	OrganisationID  string            `json:"organisation_id"`
}

func (sc *SlashCommand) CreateSlashCommand(db *gorm.DB) (SlashCommand, error) {
	err := postgresql.CreateOneRecord(db, &sc)
	if err != nil {
		return *sc, fmt.Errorf("failed to create slash command: %v", err)
	}
	return *sc, nil
}

func (sc *SlashCommand) GetIntegrationSlashCommands(db *gorm.DB, ids map[string]string) ([]SlashCommand, error) {
	var (
		slashCommands []SlashCommand
		organisation  Organisation
		integration   Integrations
	)
	exists := postgresql.CheckExists(db, &organisation, "id = ?", ids["org_id"])
	if !exists {
		return nil, fmt.Errorf("organisation does not exist")
	}

	exists = postgresql.CheckExists(db, &integration, "id = ?", ids["agent_id"])
	if !exists {
		return nil, fmt.Errorf("integration does not exist")
	}

	err := postgresql.SelectAllFromDb(db, "", &slashCommands, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if err != nil {
		return nil, fmt.Errorf("failed to get slash commands: %v", err)
	}
	return slashCommands, nil
}

func (sc *SlashCommand) GetAllOrgSlashCommands(db *gorm.DB, orgID string) ([]SlashCommand, error) {
	var (
		slashCommands []SlashCommand
	)
	err := postgresql.SelectAllFromDb(db, "", &slashCommands, "org_id = ?", orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slash commands: %v", err)
	}
	return slashCommands, nil
}

func (sc *SlashCommand) UpdateSlashCommand(db *gorm.DB, ids map[string]string, req UpdateSlashCommandRequest) (SlashCommand, error) {

	exists := postgresql.CheckExists(db, &sc, "org_id = ? AND integration_id = ? AND id = ?", ids["org_id"], ids["agent_id"], ids["command_id"])
	if !exists {
		return *sc, fmt.Errorf("slash command does not exist")
	}

	record, err := postgresql.UpdateFields(db, &sc, req, "id = ?", sc.ID)
	if err != nil {
		return *sc, fmt.Errorf("failed to update slash command: %v", err)
	}
	if record.RowsAffected == 0 {
		return *sc, fmt.Errorf("no record was updated")
	}

	return *sc, nil
}

func (sc *SlashCommand) DeleteSlashCommand(db *gorm.DB, ids map[string]string) error {
	exists := postgresql.CheckExists(db, &sc, "org_id = ? AND integration_id = ?", ids["org_id"], ids["agent_id"])
	if !exists {
		return fmt.Errorf("slash command does not exist")
	}

	err := postgresql.DeleteRecordFromDb(db, &sc)
	if err != nil {
		return fmt.Errorf("failed to delete slash command: %v", err)
	}
	return nil
}

func (sc *SlashCommand) GetDefaultSlashCommands(db *gorm.DB) ([]SlashCommand, error) {
	var (
		slashCommands []SlashCommand
	)
	err := postgresql.SelectAllFromDb(db, "", &slashCommands, "is_default = ?", true)
	if err != nil {
		return nil, fmt.Errorf("failed to get default slash commands: %v", err)
	}
	return slashCommands, nil
}
