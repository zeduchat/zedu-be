package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type AgentSkill struct {
	ID           string    `gorm:"column:id;type:uuid" json:"id"`
	Name         string    `gorm:"column:name;type:text" json:"name"`
	AgentId      string    `gorm:"column:agent_id;type:uuid" json:"agent_id"`
	Description  string    `gorm:"type:text" json:"description"`
	Type         string    `gorm:"type:text" json:"type"` // e.g MCP, A2A etc
	IsActive     bool      `gorm:"type:boolean" json:"is_active"`
	IsConfigured bool      `gorm:"type:boolean" json:"is_configured"`
	Avatar       string    `gorm:"type:text" json:"avatar"`
	Tags         []string  `gorm:"type:text[]" json:"tags"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	Config       JSONBMap  `json:"agent_config"`
}

type GeneralAgentSkill struct {
	ID           string    `gorm:"column:id;type:uuid" json:"id"`
	Name         string    `gorm:"column:name;type:text" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	Type         string    `gorm:"type:text" json:"type"` // e.g MCP, A2A etc
	IsActive     bool      `gorm:"type:boolean" json:"is_active"`
	IsConfigured bool      `gorm:"type:boolean" json:"is_configured"`
	Avatar       string    `gorm:"type:text" json:"avatar"`
	Tags         []string  `gorm:"type:text[]" json:"tags"`
	Link         string    `gorm:"type:text" json:"link"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	Config       JSONBMap  `json:"agent_config"`
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
	SkillIds    []string `json:"skill_ids" validate:"dive,uuid"`
}

type UpdateAgentSkillRequest struct {
	Config   JSONBMap `json:"agent_config"`
	SkillId  string   `json:"skill_id"`
	AgentId  string   `json:"agent_id"`
	IsActive bool     `json:"is_acive"`
}

type CreateAgentSkillsRequest struct {
	AgentId  string   `json:"agent_id"`
	SkillIds []string `json:"skill_ids" validate:"required,dive,uuid"`
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

func (a *GeneralAgentSkill) GetGeneralAgentSkills(db *gorm.DB, c *gin.Context) ([]GeneralAgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []GeneralAgentSkill

	pagination := postgresql.GetPagination(c)
	query := db.Model(&GeneralAgentSkill{})

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

func (a *AgentSkill) GetAgentSkillByID(db *gorm.DB) (AgentSkill, error) {
	var skill AgentSkill
	err := db.Where("id = ? AND agent_id = ?", a.ID, a.AgentId).First(&skill).Error
	return skill, err
}

func (a *GeneralAgentSkill) GetGeneralAgentSkillByID(db *gorm.DB, id string) (GeneralAgentSkill, error) {
	var skill GeneralAgentSkill
	err := db.Where("id = ?", id).First(&skill).Error
	return skill, err
}

func (a *AgentSkill) UpdateAgentSkill(db *gorm.DB, updateData UpdateAgentSkillRequest) (AgentSkill, error) {
	var skill AgentSkill
	exists := postgresql.CheckExists(db, &skill, "id = ? AND agent_id = ?", a.ID, a.AgentId)
	if !exists {
		return skill, errors.New("agent skill not found")
	}

	result, err := postgresql.UpdateFields(db, &skill, updateData, "id = ? AND agent_id = ?", a.ID, a.AgentId)
	if err != nil {
		return skill, errors.New("failed to update agent skill")
	}
	if result.RowsAffected == 0 {
		return skill, errors.New("no record updated")
	}

	_ = db.Where("id = ? AND agent_id = ?", a.ID, a.AgentId).First(&skill).Error
	return skill, nil
}

func (a *AgentSkill) DeleteAgentSkill(db *gorm.DB) error {
	var skill AgentSkill
	exists := postgresql.CheckExists(db, &skill, "id = ? AND agent_id = ?", a.ID, a.AgentId)
	if !exists {
		return errors.New("agent skill not found")
	}
	return db.Delete(&skill, "id = ? AND agent_id = ?", a.ID, a.AgentId).Error
}

func (a *AgentSkill) ValidateSkills(db *gorm.DB, req *CreateAgentSkillsRequest) error {
	invalidSkill := []string{}

	if len(req.SkillIds) == 0 {
		return fmt.Errorf("Invalid skills supplied, empty skills")
	}

	for _, skillId := range req.SkillIds {
		exists := postgresql.CheckExists(db, &GeneralAgentSkill{}, "id = ?", req.SkillIds)
		if !exists {
			invalidSkill = append(invalidSkill, skillId)
		}
	}

	return fmt.Errorf("Invalid Skills supplied: %v", invalidSkill)
}

func (a *AgentSkill) AddSkilltoAgent(db *gorm.DB, req *CreateAgentSkillsRequest) error {

	skills := []AgentSkill{}

	for _, skillId := range req.SkillIds {
		skills = append(skills, AgentSkill{
			ID:       skillId,
			AgentId:  req.AgentId,
			IsActive: true,
		})
	}

	err := postgresql.CreateMultipleRecords(db, &skills, len(skills))

	if err != nil {
		return errors.New("An error occurred adding skills to agent")
	}

	return nil
}
