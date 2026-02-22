package models

type IDS struct {
	OrganisationID string `json:"organisation_id"`
	AgentID        string `json:"agent_id"`
	UserID         string `json:"user_id"`
	ChannelID      string `json:"channel_id"`
	TaskID         string `json:"task_id"`
	SkillID        string `json:"skill_id"`
	SettingID      string `json:"setting_id"`
	RoleID         string `json:"role_id"`
	OrgRoleID      string `json:"org_role_id"`
	MessageID      string `json:"message_id"`
	ThreadID       string `json:"thread_id"`
	ReactionID     string `json:"reaction_id"`
	OwnerID        string `json:"owner_id"`
	Type           string `json:"type"`
	WorkflowID     string `json:"workflow_id"`
	Search         string `json:"search"`
}
