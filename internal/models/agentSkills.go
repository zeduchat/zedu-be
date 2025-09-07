package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type AgentSkill struct {
	ID           string         `gorm:"column:id;type:uuid" json:"-"`
	Name         string         `gorm:"column:name;type:text" json:"name"`
	AgentId      string         `gorm:"column:agent_id;type:uuid" json:"agent_id"`
	SkillId      string         `gorm:"column:skill_id;type:uuid" json:"skill_id"`
	Description  string         `gorm:"type:text" json:"description"`
	Type         string         `gorm:"type:text" json:"type"` // e.g MCP, A2A etc
	IsActive     bool           `gorm:"type:boolean" json:"is_active"`
	IsConfigured bool           `gorm:"type:boolean" json:"is_configured"`
	Avatar       string         `gorm:"type:text" json:"avatar"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	Config       JSONBMapArr    `gorm:"type:jsonb" json:"agent_config"`
	Link         string         `gorm:"type:text" json:"-"`
	Tags         pq.StringArray `gorm:"type:text[]" json:"tags"`
	UserId       string         `gorm:"type:uuid" json:"-"`
	OrgId        string         `gorm:"type:uuid" json:"-"`
}

type GeneralAgentSkill struct {
	ID           string         `gorm:"column:id;type:uuid" json:"id"`
	Name         string         `gorm:"column:name;type:text" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	Type         string         `gorm:"type:text" json:"type"` // e.g MCP, A2A etc
	IsActive     bool           `gorm:"type:boolean" json:"is_active"`
	IsConfigured bool           `gorm:"type:boolean" json:"is_configured"`
	Avatar       string         `gorm:"type:text" json:"avatar"`
	Tags         pq.StringArray `gorm:"type:text[]" json:"tags"`
	Link         string         `gorm:"type:text" json:"-"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	Config       JSONBMapArr    `gorm:"type:jsonb" json:"agent_config"`
}

type CreateAgentSkillRequest struct {
	Name        string      `json:"name" validate:"required"`
	Description string      `json:"description" validate:"required"`
	Type        string      `json:"type" validate:"required,oneof=MCP A2A"`
	Config      JSONBMapArr `json:"agent_config"`
	AgentId     string      `json:"agent_id" validate:"required"`
	IsActive    bool        `json:"is_acive"`
	URLLink     string      `json:"url_link" validate:"required"`
	Avatar      string      `json:"avatar"`
	Tags        []string    `json:"tags"`
	OrgId       string      `json:"-"`
	UserId      string      `json:"-"`
	SkillId     string      `json:"-"`
}

type UpdateAgentSkillRequest struct {
	Config   JSONBMapArr `json:"agent_config"`
	SkillId  string      `json:"skill_id"`
	AgentId  string      `json:"agent_id"`
	IsActive bool        `json:"is_active"`
	OrgId    string      `json:"-"`
	UserId   string      `json:"-"`
}

type CreateAgentSkillsRequest struct {
	AgentId  string   `json:"agent_id"`
	SkillIds []string `json:"skill_ids" validate:"required,dive,uuid"`
	OrgId    string   `json:"-"`
	UserId   string   `json:"-"`
}

type AgentSkillResponse struct {
	SkillId      string      `json:"skill_id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"`
	IsActive     bool        `json:"is_active"`
	IsConfigured bool        `json:"is_configured"`
	Avatar       string      `json:"avatar"`
	Config       JSONBMapArr `json:"agent_config"`
	Tags         []string    `json:"tags"`
}

type SkillResp struct {
	AgentID string
	SkillID string
}

type JSONBMapArr []JSONBMap

func (j *JSONBMapArr) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONBMapArr) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (a *AgentSkill) CreateAgentSkill(db *gorm.DB) error {
	return postgresql.CreateOneRecord(db, a)
}

func (as *AgentSkill) CheckAgentHasSkill(db *gorm.DB, agentID, skillID string) (bool, error) {
	var skill AgentSkill
	err := db.Where("agent_id = ? AND skill_id = ?", agentID, skillID).First(&skill).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (as *AgentSkill) CheckAgentHasSkillByName(db *gorm.DB, agentID, skillName string) (bool, error) {
	var skill AgentSkill
	err := db.Where("agent_id = ? AND name = ?", agentID, skillName).First(&skill).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *AgentSkill) GetAgentSkills(db *gorm.DB, c *gin.Context) ([]AgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []AgentSkill

	pagination := postgresql.GetPagination(c)
	query := db.Model(&AgentSkill{}).
		Select(`
		agent_skills.skill_id, 
		agent_skills.agent_id, 
		agent_skills.config,
		agent_skills.is_configured,
		agent_skills.created_at,
		agent_skills.is_active,
		COALESCE(general_agent_skills.name, agent_skills.name) AS name,
		COALESCE(general_agent_skills.tags, agent_skills.tags) AS tags,
		COALESCE(general_agent_skills.type, agent_skills.type) AS type,
		COALESCE(general_agent_skills.description, agent_skills.description) AS description,
		COALESCE(general_agent_skills.avatar, agent_skills.avatar) AS avatar
	`).
		Joins("LEFT JOIN general_agent_skills ON general_agent_skills.id = agent_skills.skill_id").
		Where("agent_skills.agent_id = ? AND agent_skills.org_id = ?", a.AgentId, a.OrgId)
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

func (a *GeneralAgentSkill) FetchGeneralAgentSkills(db *gorm.DB, c *gin.Context) ([]GeneralAgentSkill, error, int) {
	var skills []GeneralAgentSkill

	query := db.Model(&GeneralAgentSkill{})

	err := postgresql.SelectAllFromDb(
		query,
		"desc",
		&skills,
		nil,
	)
	if err != nil {
		return skills, errors.New("failed to fetch general skills"), http.StatusInternalServerError
	}

	return skills, nil, http.StatusOK
}

func (a *AgentSkill) GetAgentSkillByID(db *gorm.DB) (AgentSkillResponse, error) {
	var skill AgentSkillResponse

	err := db.Model(&AgentSkill{}).
		Select(`
        agent_skills.skill_id, 
        agent_skills.agent_id, 
        agent_skills.config,
        agent_skills.is_configured,
        agent_skills.created_at,
        agent_skills.is_active,
        COALESCE(general_agent_skills.name, agent_skills.name) AS name,
        COALESCE(general_agent_skills.tags, agent_skills.tags) AS tags,
        COALESCE(general_agent_skills.type, agent_skills.type) AS type,
        COALESCE(general_agent_skills.description, agent_skills.description) AS description,
        COALESCE(general_agent_skills.avatar, agent_skills.avatar) AS avatar
        `).
		Joins("LEFT JOIN general_agent_skills ON general_agent_skills.id = agent_skills.skill_id").
		Where("agent_skills.agent_id = ? AND agent_skills.skill_id = ? AND agent_skills.org_id = ?", a.AgentId, a.SkillId, a.OrgId).
		First(&skill).Error

	if err != nil {
		return skill, errors.New("Skill not binded to agent")
	}

	return skill, nil
}

func (a *AgentSkill) GetAllAgentSkills(db *gorm.DB) ([]AgentSkillResponse, error) {
	var skills []AgentSkillResponse

	err := db.Table("agent_skills").
		Select(`
		agent_skills.skill_id, 
		agent_skills.agent_id, 
		agent_skills.config,
		agent_skills.is_configured,
		agent_skills.created_at,
		agent_skills.is_active,
		COALESCE(general_agent_skills.name, agent_skills.name) AS name,
		COALESCE(general_agent_skills.tags, agent_skills.tags) AS tags,
		COALESCE(general_agent_skills.type, agent_skills.type) AS type,
		COALESCE(general_agent_skills.description, agent_skills.description) AS description,
		COALESCE(general_agent_skills.avatar, agent_skills.avatar) AS avatar
	`).
		Joins("LEFT JOIN general_agent_skills ON general_agent_skills.id = agent_skills.skill_id").
		Where("agent_skills.agent_id = ? AND agent_skills.org_id = ?", a.AgentId, a.OrgId).
		Scan(&skills).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent skills: %w", err)
	}

	if len(skills) == 0 {
		return []AgentSkillResponse{}, nil
	}

	return skills, nil
}

func (a *GeneralAgentSkill) GetGeneralAgentSkillByID(db *gorm.DB, id string) error {
	err := db.Where("id = ?", id).First(&a).Error
	return err
}

func (a *AgentSkill) UpdateAgentSkill(db *gorm.DB, updateData UpdateAgentSkillRequest) (AgentSkill, error) {
	var skill AgentSkill
	exists := postgresql.CheckExists(db, &skill, "skill_id = ? AND agent_id = ? AND org_id = ?", a.SkillId, a.AgentId, a.OrgId)
	if !exists {
		return skill, errors.New("agent skill not found")
	}

	updates := map[string]any{
		"is_active": updateData.IsActive,
		"config":    updateData.Config,
	}

	result, err := postgresql.UpdateFields(db, &skill, updates, "skill_id = ? AND agent_id = ?", a.SkillId, a.AgentId)
	if err != nil {
		return skill, errors.New("failed to update agent skill")
	}
	if result.RowsAffected == 0 {
		return skill, errors.New("no record updated")
	}

	return skill, nil
}

func (a *AgentSkill) DeleteAgentSkill(db *gorm.DB) error {
	var skill AgentSkill
	exists := postgresql.CheckExists(db, &skill, "skill_id = ? AND agent_id = ? AND org_id = ?", a.SkillId, a.AgentId, a.OrgId)
	if !exists {
		return errors.New("agent skill not found")
	}
	return db.Delete(&skill, "skill_id = ? AND agent_id = ?", a.SkillId, a.AgentId).Error
}

func (a *AgentSkill) ValidateSkills(db *gorm.DB, req *CreateAgentSkillsRequest) error {
	invalidSkill := []string{}
	validSkillIds := []string{}
	var org_integration OrganisationIntegrations

	if len(req.SkillIds) == 0 {
		return fmt.Errorf("invalid skills supplied, empty skills")
	}

	exists := postgresql.CheckExists(db, &org_integration, "integration_id = ?", req.AgentId)
	if !exists {
		return errors.New("agent does not exist")
	}

	for _, skillId := range req.SkillIds {

		exists := postgresql.CheckExists(db, &GeneralAgentSkill{}, "id = ?", skillId)
		if !exists {
			invalidSkill = append(invalidSkill, skillId)
			continue
		}
		exists = postgresql.CheckExists(db, &AgentSkill{}, "skill_id = ? AND agent_id = ?", skillId, req.AgentId)
		if exists {
			continue
		}

		validSkillIds = append(validSkillIds, skillId)
	}

	req.SkillIds = validSkillIds

	if len(invalidSkill) > 0 {
		return fmt.Errorf("invalid skills supplied: %v", invalidSkill)
	}

	return nil
}

func (a *AgentSkill) AddSkilltoAgent(db *gorm.DB, req *CreateAgentSkillsRequest) error {

	skills := []AgentSkill{}

	if len(req.SkillIds) == 0 {
		return nil
	}

	for _, skillId := range req.SkillIds {
		skills = append(skills, AgentSkill{
			ID:       utility.GenerateUUID(),
			SkillId:  skillId,
			AgentId:  req.AgentId,
			IsActive: true,
			OrgId:    req.OrgId,
			UserId:   req.UserId,
		})
	}

	err := postgresql.CreateMultipleRecords(db, &skills, len(skills))

	if err != nil {
		return err
	}

	return nil
}
