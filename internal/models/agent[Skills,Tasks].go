package models

type AgentSkill struct {
	ID           string   `gorm:"column:id;type:uuid" json:"id"`
	Name         string   `gorm:"column:name;type:text" json:"name"`
	AgentId      string   `gorm:"column:agent_id;type:uuid" json:"agent_id"`
	Description  string   `gorm:"type:text" json:"description"`
	Type         string   `gorm:"type:text" json:"type"` // e.g MCP, A2A etc
	IsActive     bool     `gorm:"type:boolean" json:"is_active"`
	IsConfigured bool     `gorm:"type:boolean" json:"is_configured"`
	Avatar       string   `gorm:"type:text" json:"avatar"`
	Config       JSONBMap `json:"agent_config"`
	Tags         []string `json:"tags"`
}

type SkillConfig map[string]any

type CreateAgentSkillRequest struct {
	Name        string      `json:"name" binding:"required"`
	Description string      `json:"description" binding:"required"`
	Type        string      `json:"type" binding:"required"`
	IsActive    bool        `json:"is_active" binding:"required"`
	Config      SkillConfig `json:"agent_config" binding:"required"`
	Tags        []string    `json:"tags" binding:"required"`
	JSONUrl     string      `json:"json_url" binding:"required,url"`
}

type AddSkillToAgentRequest struct {
	SkillId string `json:"skill_id"`
	AgentId string `json:"agent_id"`
}

type AgentSkillResponse struct{}
