package models

type AgentSkill struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"` // e.g MCP, A2A etc
	IsActive    bool        `json:"is_active"`
	Config      SkillConfig `json:"agent_config"`
	Tags        []string    `json:"tags"`
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

