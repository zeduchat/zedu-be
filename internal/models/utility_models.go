package models

type IDS struct {
	OrganisationID string `json:"organisation_id"`
	AgentID        string `json:"agent_id"`
	UserID         string `json:"user_id"`
	ChannelID      string `json:"channel_id"`
	SettingID      string `json:"setting_id"`
	RoleID         string `json:"role_id"`
	OrgRoleID      string `json:"org_role_id"`
}
