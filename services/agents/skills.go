package agents

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

// Create a new AgentSkill
func CreateAgentSkill(req models.CreateAgentSkillRequest, db *gorm.DB, logger *utility.Logger) (models.AgentSkillResponse, int, error) {
	var resp models.AgentSkillResponse

	agentExists := postgresql.CheckExists(db, &models.OrganisationIntegrations{}, "integration_id = ?", req.AgentId)
	if !agentExists {
		return resp, http.StatusNotFound, errors.New("agent not found")
	}

	agentSkillExists := postgresql.CheckExists(db, &models.AgentSkill{}, "agent_id = ?", req.AgentId)
	if agentSkillExists {
		return resp, http.StatusBadRequest, errors.New("agent already has the skill")
	}

	// add to general Agent skill

	genAgentSkill := models.GeneralAgentSkill{
		ID:           utility.GenerateUUID(),
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		IsActive:     true,
		IsConfigured: false, // default
		Avatar:       req.Avatar,
		Config:       req.Config,
		Link:         req.URLLink,
		Tags:         req.Tags,
		Category:     req.Category,
	}

	if err := genAgentSkill.CreateGeneralAgentSkill(db); err != nil {
		return resp, http.StatusInternalServerError, err
	}

	agentSkill := models.AgentSkill{
		ID:           utility.GenerateUUID(),
		SkillId:      genAgentSkill.ID,
		Name:         req.Name,
		AgentId:      req.AgentId,
		Description:  req.Description,
		Type:         req.Type,
		IsActive:     true,
		IsConfigured: false, // default
		Avatar:       req.Avatar,
		Config:       req.Config,
		Link:         req.URLLink,
		Tags:         req.Tags,
		OrgId:        req.OrgId,
		UserId:       req.UserId,
		Category:     req.Category,
	}

	if err := agentSkill.CreateAgentSkill(db); err != nil {
		return resp, http.StatusInternalServerError, err
	}

	resp = models.AgentSkillResponse{
		SkillId:      agentSkill.SkillId,
		Name:         agentSkill.Name,
		Description:  agentSkill.Description,
		Type:         agentSkill.Type,
		IsActive:     agentSkill.IsActive,
		IsConfigured: agentSkill.IsConfigured,
		Avatar:       agentSkill.Avatar,
		Config:       agentSkill.Config,
		Tags:         agentSkill.Tags,
		Category:     agentSkill.Category,
	}

	return resp, http.StatusCreated, nil
}

func GetAgentSkills(req models.CreateAgentSkillRequest, db *gorm.DB, c *gin.Context) ([]models.AgentSkill, postgresql.PaginationResponse, error, int) {
	var skill models.AgentSkill
	skill.AgentId = req.AgentId
	skill.OrgId = req.OrgId
	return skill.GetAgentSkills(db, c)
}

func GetAgentSkillByID(req models.CreateAgentSkillRequest, db *gorm.DB) (models.AgentSkillResponse, error) {
	var skill models.AgentSkill
	skill.AgentId = req.AgentId
	skill.SkillId = req.SkillId
	skill.OrgId = req.OrgId
	return skill.GetAgentSkillByID(db)
}

func GetGeneralAgentSkills(db *gorm.DB, c *gin.Context) ([]models.GeneralAgentSkill, postgresql.PaginationResponse, error, int) {
	var skill models.GeneralAgentSkill
	return skill.GetGeneralAgentSkills(db, c)
}

func GetGeneralAgentSkillByID(skillID string, db *gorm.DB) (models.GeneralAgentSkill, error) {
	var skill models.GeneralAgentSkill
	err := skill.GetGeneralAgentSkillByID(db, skillID)

	if err != nil {
		return skill, errors.New("skill does not exists")
	}

	return skill, nil
}

func UpdateAgentSkill(req models.CreateAgentRequest, updateData models.UpdateAgentSkillRequest, db *gorm.DB) (models.AgentSkill, error) {
	var skill models.AgentSkill
	skill.SkillId = req.SkillId
	skill.AgentId = req.AgentId
	skill.OrgId = req.OrgId
	skill.UserId = req.UserId
	return skill.UpdateAgentSkill(db, updateData)
}

func DeleteAgentSkill(req models.CreateAgentRequest, db *gorm.DB) error {
	var skill models.AgentSkill
	skill.SkillId = req.SkillId
	skill.AgentId = req.AgentId
	skill.OrgId = req.OrgId
	skill.UserId = req.SkillId
	return skill.DeleteAgentSkill(db)
}

func AddSkillToAgent(req models.CreateAgentSkillsRequest, db *gorm.DB, logger *utility.Logger) (int, error) {
	var skill models.AgentSkill

	// all or nothing validation
	err := skill.ValidateSkills(db, &req)
	if err != nil {
		return http.StatusBadRequest, err
	}

	err = skill.AddSkilltoAgent(db, &req)

	if err != nil {
		logger.Error("Error adding skills to agent an error occured: %v", err)
		return http.StatusInternalServerError, errors.New("An error occurred adding skills to agent")
	}

	return http.StatusOK, nil
}
