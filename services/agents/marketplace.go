package agents

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
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
	categories := c.QueryArray("category")
	popular := c.Query("popular")
	sortBy := c.Query("sort_by")

	var (
		pagination postgresql.PaginationResponse
		err        error
		code       int
	)

	switch {
	case len(categories) > 0:
		resp, pagination, err, code = agents.GetAgentsByCategory(db, c, categories, sortBy)
	case search != "":
		resp, pagination, err, code = agents.SearchAgents(db, c, search, sortBy)
	case popular == "true":
		resp, pagination, err, code = agents.GetPopularAgents(db, c, sortBy)
	default:
		resp, pagination, err, code = agents.GetSystemAgentApps(db, c, sortBy)
	}

	if err != nil {
		return &[]models.AgentResp{}, pagination, err, code
	}

	// Format response
	for _, a := range resp {

		parts := strings.Split(a.ID, "-")
		lastPart := parts[len(parts)-1]

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
			AgentSlug:   fmt.Sprintf("%s-%s", slug.Make(a.Name), lastPart),
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
	categories := c.QueryArray("category")
	sortBy := c.Query("sort_by")

	var (
		pagination postgresql.PaginationResponse
		err        error
		code       int
	)

	switch {
	case len(categories) > 0:
		resp, pagination, err, code = skill.GetSkillsByCategory(db, c, categories, sortBy)
	case search != "":
		resp, pagination, err, code = skill.SearchSkills(db, c, search, sortBy)
	default:
		resp, pagination, err, code = skill.GetGeneralAgentSkills(db, c)
	}

	if err != nil {
		return &[]models.AgentSkillResponse{}, pagination, err, code
	}

	for _, s := range resp {

		parts := strings.Split(s.ID, "-")
		lastPart := parts[len(parts)-1]

		skillResp = append(skillResp, models.AgentSkillResponse{
			SkillId:          s.ID,
			Name:             s.Name,
			Description:      s.Description,
			Type:             s.Type,
			IsActive:         s.IsActive,
			IsConfigured:     s.IsConfigured,
			Avatar:           s.Avatar,
			Config:           s.Config,
			Tags:             s.Tags,
			Category:         s.Category,
			ShortDescription: s.ShortDescription,
			LongDescription:  s.LongDescription,
			SkillSlug:        fmt.Sprintf("%s-%s", slug.Make(s.Name), lastPart),
		})
	}

	return &skillResp, pagination, nil, code
}

// Workflow

func SearchWorkflowsService(c *gin.Context, db *gorm.DB) (*[]models.GeneralWorkflow, postgresql.PaginationResponse, error, int) {
	var (
		workflow models.GeneralWorkflow
		resp     *[]models.GeneralWorkflow
	)

	search := c.Query("search")
	categories := c.QueryArray("category")
	sortBy := c.Query("sort_by")

	var (
		pagination postgresql.PaginationResponse
		err        error
		code       int
	)

	switch {
	case len(categories) > 0:
		resp, pagination, err, code = workflow.GetWorkflowsByCategory(db, c, categories, sortBy)
	case search != "":
		resp, pagination, err, code = workflow.SearchWorkflows(db, c, search, sortBy)
	default:
		resp, pagination, err = workflow.GetMarketPlaceWorkflows(db, c)
		if err != nil {
			code = http.StatusOK
		} else {
			code = http.StatusInternalServerError
		}
	}

	if err != nil {
		return &[]models.GeneralWorkflow{}, pagination, err, code
	}
	return resp, pagination, nil, code
}

// Categories
func GetAgentCategories(c *gin.Context, db *gorm.DB) (gin.H, error) {

	agentCategories, err := models.GetUniqueAgentCategories(db)
	if err != nil {
		return nil, err
	}

	workflowCategories, err := models.GetUniqueWorkflowCategories(db)
	if err != nil {
		return nil, err
	}

	skillCategories, err := models.GetUniqueSkillsCategories(db)
	if err != nil {
		return nil, err
	}

	response := gin.H{
		"agents":   agentCategories,
		"workflow": workflowCategories,
		"skills":   skillCategories,
	}

	return response, nil
}

func GetSkillCategories(c *gin.Context, db *gorm.DB) (gin.H, error) {

	skillCategories, err := models.GetUniqueSkillsCategories(db)
	if err != nil {
		return nil, err
	}

	response := gin.H{
		"skills": skillCategories,
	}

	return response, nil
}

func GetWorkflowCategories(c *gin.Context, db *gorm.DB) (gin.H, error) {

	workflowCategories, err := models.GetUniqueWorkflowCategories(db)
	if err != nil {
		return nil, err
	}

	response := gin.H{
		"workflow": workflowCategories,
	}

	return response, nil
}
