package agents

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

// Agents

func SearchAgentsService(c *gin.Context, db *gorm.DB) (*[]models.AgentResp, postgresql.PaginationResponse, error, int) {
	var (
		agents  models.Integrations
		resp    []models.Integrations
		botResp []models.AgentResp = make([]models.AgentResp, 0)
	)

	// Get query params
	search := c.Query("search")
	category := c.Query("category")
	featured := c.Query("featured")

	var (
		pagination postgresql.PaginationResponse
		err        error
		code       int
	)

	switch {
	case featured == "true":
		resp, pagination, err, code = agents.GetFeaturedAgents(db, c)
	case category != "":
		resp, pagination, err, code = agents.GetAgentsByCategory(db, c, category)
	case search != "":
		resp, pagination, err, code = agents.SearchAgents(db, c, search)
	default:
		return &[]models.AgentResp{}, postgresql.PaginationResponse{}, errors.New("invalid search query"), http.StatusBadRequest
	}

	if err != nil {
		return &[]models.AgentResp{}, pagination, err, code
	}

	// Format response
	for _, a := range resp {
		agent := models.AgentResp{
			ID:          a.ID,
			Name:        a.Name,
			Title:       a.Title,
			Tone:        a.Tone,
			Visibility:  a.Visibility,
			Avatar:      a.AppLogo,
			Description: a.AppDescription,
			IsActive:    a.IsActive,
			Category:    a.Category,
			Stars:       a.Stars,
		}
		botResp = append(botResp, agent)
	}

	return &botResp, pagination, nil, code
}

// Skills

func SearchSkillsService(c *gin.Context, db *gorm.DB) (*[]models.AgentSkillResponse, postgresql.PaginationResponse, error, int) {
	var (
		skill     models.GeneralAgentSkill
		resp      []models.GeneralAgentSkill
		skillResp []models.AgentSkillResponse = make([]models.AgentSkillResponse, 0)
	)

	search := c.Query("search")
	category := c.Query("category")

	var (
		pagination postgresql.PaginationResponse
		err        error
		code       int
	)

	switch {
	case category != "":
		resp, pagination, err, code = skill.GetSkillsByCategory(db, c, category)
	case search != "":
		resp, pagination, err, code = skill.SearchSkills(db, c, search)
	default:
		return &[]models.AgentSkillResponse{}, postgresql.PaginationResponse{}, errors.New("invalid search query"), http.StatusBadRequest
	}

	if err != nil {
		return &[]models.AgentSkillResponse{}, pagination, err, code
	}

	for _, s := range resp {
		skillResp = append(skillResp, models.AgentSkillResponse{
			SkillId:      s.ID,
			Name:         s.Name,
			Description:  s.Description,
			Type:         s.Type,
			IsActive:     s.IsActive,
			IsConfigured: s.IsConfigured,
			Avatar:       s.Avatar,
			Config:       s.Config,
			Tags:         s.Tags,
			Category:     s.Category,
		})
	}

	return &skillResp, pagination, nil, code
}

// Workflow

func SearchWorkflowsService(c *gin.Context, db *gorm.DB) (*[]models.GeneralWorkflow, postgresql.PaginationResponse, error, int) {
	var (
		workflow models.GeneralWorkflow
		resp     []models.GeneralWorkflow
	)

	search := c.Query("search")
	category := c.Query("category")

	var (
		pagination postgresql.PaginationResponse
		err        error
		code       int
	)

	switch {
	case category != "":
		resp, pagination, err, code = workflow.GetWorkflowsByCategory(db, c, category)
	case search != "":
		resp, pagination, err, code = workflow.SearchWorkflows(db, c, search)
	default:
		return &[]models.GeneralWorkflow{}, postgresql.PaginationResponse{}, errors.New("invalid search query"), http.StatusBadRequest
	}

	if err != nil {
		return &[]models.GeneralWorkflow{}, pagination, err, code
	}
	return &resp, pagination, nil, code
}

// Categories

func GetAllCategories(c *gin.Context, db *gorm.DB) (gin.H, error) {
	// Fetch agent categories
	agentCategories, err := models.GetUniqueCategories(db)
	if err != nil {
		return nil, err
	}

	// Fetch workflow categories
	workflowCategories, err := models.GetUniqueWorkflowCategories(db)
	if err != nil {
		return nil, err
	}

	// Fetch skill categories
	skillCategories, err := models.GetUniqueSkillsCategories(db)
	if err != nil {
		return nil, err
	}

	// Build response
	response := gin.H{
		"agents":   agentCategories,
		"workflow": workflowCategories,
		"skills":   skillCategories,
	}

	return response, nil
}
