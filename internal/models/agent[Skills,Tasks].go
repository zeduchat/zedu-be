package models

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

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

type GeneralAgentSkill struct {
	ID           string   `gorm:"column:id;type:uuid" json:"id"`
	Name         string   `gorm:"column:name;type:text" json:"name"`
	Description  string   `gorm:"type:text" json:"description"`
	Type         string   `gorm:"type:text" json:"type"` // e.g MCP, A2A etc
	IsActive     bool     `gorm:"type:boolean" json:"is_active"`
	IsConfigured bool     `gorm:"type:boolean" json:"is_configured"`
	Avatar       string   `gorm:"type:text" json:"avatar"`
	Config       JSONBMap `json:"agent_config"`
	Tags         []string `json:"tags"`
}
type CreateAgentSkillRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Type        string   `json:"type" validate:"required"`
	Config      JSONBMap `json:"agent_config"`
	SkillId     string   `json:"skill_id" validate:"required"`
	AgentId     string   `json:"agent_id" validate:"required"`
	IsActive    bool     `json:"is_acive"`
	Tags        []string `json:"tags"`
}

type AgentSkillResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	IsActive     bool     `json:"is_active"`
	IsConfigured bool     `json:"is_configured"`
	Avatar       string   `json:"avatar"`
	Config       JSONBMap `json:"agent_config"`
	Tags         []string `json:"tags"`
}

func (a *AgentSkill) CreateAgentSkill(db *gorm.DB) error {
	return postgresql.CreateOneRecord(db, a)
}

func (a *AgentSkill) GetAgentSkills(db *gorm.DB, agentID string, c *gin.Context) ([]AgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []AgentSkill

	pagination := postgresql.GetPagination(c)
	query := db.Model(&AgentSkill{}).Where("agent_id = ?", agentID)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&skills,
		nil,
	)
	if err != nil {
		return skills, paginationResponse, err, http.StatusInternalServerError
	}

	return skills, paginationResponse, nil, http.StatusOK
}

func (a *AgentSkill) GetAgentSkillByID(db *gorm.DB, id string) (AgentSkill, error) {
	var skill AgentSkill
	err := db.Where("id = ?", id).First(&skill).Error
	return skill, err
}

func (a *AgentSkill) UpdateAgentSkill(db *gorm.DB, id string, updateData map[string]interface{}) (AgentSkill, error) {
	var skill AgentSkill
	exists := postgresql.CheckExists(db, &skill, "id = ?", id)
	if !exists {
		return skill, errors.New("agent skill not found")
	}

	result, err := postgresql.UpdateFields(db, &skill, updateData, "id = ?", id)
	if err != nil {
		return skill, errors.New("failed to update agent skill")
	}
	if result.RowsAffected == 0 {
		return skill, errors.New("no record updated")
	}

	_ = db.Where("id = ?", id).First(&skill).Error
	return skill, nil
}

func (a *AgentSkill) DeleteAgentSkill(db *gorm.DB, id string) error {
	var skill AgentSkill
	exists := postgresql.CheckExists(db, &skill, "id = ?", id)
	if !exists {
		return errors.New("agent skill not found")
	}
	return db.Delete(&skill, "id = ?", id).Error
}
