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

	agentSkill := models.AgentSkill{
		ID:           utility.GenerateUUID(),
		Name:         req.Name,
		AgentId:      req.AgentId,
		Description:  req.Description,
		Type:         req.Type,
		IsActive:     req.IsActive,
		IsConfigured: false, // default
		Avatar:       "",    // can be updated later
		Config:       req.Config,
		Tags:         req.Tags,
	}

	if err := agentSkill.CreateAgentSkill(db); err != nil {
		return resp, http.StatusInternalServerError, err
	}

	resp = models.AgentSkillResponse{
		ID:           agentSkill.ID,
		Name:         agentSkill.Name,
		Description:  agentSkill.Description,
		Type:         agentSkill.Type,
		IsActive:     agentSkill.IsActive,
		IsConfigured: agentSkill.IsConfigured,
		Avatar:       agentSkill.Avatar,
		Config:       agentSkill.Config,
		Tags:         agentSkill.Tags,
	}

	return resp, http.StatusCreated, nil
}

func GetAgentSkills(agentID string, db *gorm.DB, c *gin.Context) ([]models.AgentSkill, postgresql.PaginationResponse, error, int) {
	var skill models.AgentSkill
	return skill.GetAgentSkills(db, agentID, c)
}

func GetAgentSkillByID(skillID string, db *gorm.DB) (models.AgentSkill, error) {
	var skill models.AgentSkill
	return skill.GetAgentSkillByID(db, skillID)
}

func UpdateAgentSkill(skillID string, updateData map[string]interface{}, db *gorm.DB) (models.AgentSkill, error) {
	var skill models.AgentSkill
	return skill.UpdateAgentSkill(db, skillID, updateData)
}

func DeleteAgentSkill(skillID string, db *gorm.DB) error {
	var skill models.AgentSkill
	return skill.DeleteAgentSkill(db, skillID)
}
