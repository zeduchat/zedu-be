package agents

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

// Create a new AgentSkill
func CreateAgentSkill(req models.CreateAgentSkillRequest, db *gorm.DB, logger *utility.Logger) (models.AgentSkillResponse, int, error) {
	var resp models.AgentSkillResponse

	// add to general Agent skill

	genAgentSkill := models.GeneralAgentSkill{
		ID:               utility.GenerateUUID(),
		Name:             req.Name,
		Description:      req.Description,
		Type:             req.Type,
		IsActive:         true,
		IsConfigured:     false, // default
		Avatar:           req.Avatar,
		Config:           req.Config,
		Link:             req.URLLink,
		Tags:             req.Tags,
		Category:         req.Category,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
	}

	if err := genAgentSkill.CreateGeneralAgentSkill(db); err != nil {
		return resp, http.StatusInternalServerError, err
	}

	resp = models.AgentSkillResponse{
		SkillId:          genAgentSkill.ID,
		Name:             genAgentSkill.Name,
		Description:      genAgentSkill.Description,
		Type:             genAgentSkill.Type,
		IsActive:         genAgentSkill.IsActive,
		IsConfigured:     genAgentSkill.IsConfigured,
		Avatar:           genAgentSkill.Avatar,
		Config:           genAgentSkill.Config,
		Tags:             genAgentSkill.Tags,
		Category:         genAgentSkill.Category,
		ShortDescription: req.ShortDescription,
		LongDescription:  req.LongDescription,
	}

	return resp, http.StatusCreated, nil
}

func GetAgentSkills(req models.CreateAgentSkillRequest, db *gorm.DB, c *gin.Context) (*[]models.AgentSkillResponse, postgresql.PaginationResponse, error, int) {
	var (
		skill     models.AgentSkill
		skillResp = []models.AgentSkillResponse{}
	)
	skill.AgentId = req.AgentId
	skill.OrgId = req.OrgId
	skill.IsPublic = req.IsPublic

	resp, pag, err, code := skill.GetAgentSkills(db, c)

	if err != nil {
		return &skillResp, postgresql.PaginationResponse{}, err, code
	}

	for _, s := range resp {

		parts := strings.Split(s.SkillId, "-")
		lastPart := parts[len(parts)-1]

		skillResp = append(skillResp, models.AgentSkillResponse{
			SkillId:          s.SkillId,
			Name:             s.Name,
			Description:      s.Description,
			Type:             s.Type,
			IsActive:         s.IsActive,
			IsConfigured:     s.IsConfigured,
			Avatar:           s.Avatar,
			Tags:             s.Tags,
			Category:         s.Category,
			ShortDescription: s.ShortDescription,
			LongDescription:  s.LongDescription,
			SkillSlug:        fmt.Sprintf("%s-%s", slug.Make(s.Name), lastPart),
		})
	}

	return &skillResp, pag, nil, code
}

func GetAgentSkillByID(req models.CreateAgentSkillRequest, db *gorm.DB) (*models.AgentSkillResponse, error) {
	var skill models.AgentSkill
	skill.AgentId = req.AgentId
	skill.SkillId = req.SkillId
	skill.OrgId = req.OrgId

	s, err := skill.GetAgentSkillByID(db)

	parts := strings.Split(s.SkillId, "-")
	lastPart := parts[len(parts)-1]
	s.SkillSlug = fmt.Sprintf("%s-%s", slug.Make(s.Name), lastPart)

	return &s, err
}

func GetGeneralAgentSkills(db *gorm.DB, c *gin.Context) (*[]models.AgentSkillResponse, postgresql.PaginationResponse, error, int) {
	var (
		skill     models.GeneralAgentSkill
		skillResp = []models.AgentSkillResponse{}
	)

	resp, pag, err, code := skill.GetGeneralAgentSkills(db, c)

	if err != nil {
		return &skillResp, postgresql.PaginationResponse{}, err, code
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

	return &skillResp, pag, nil, code
}

func GetGeneralAgentSkillByID(skillID string, db *gorm.DB) (*models.AgentSkillResponse, error) {
	var (
		skill     models.GeneralAgentSkill
		skillResp = models.AgentSkillResponse{}
	)
	err := skill.GetGeneralAgentSkillByID(db, skillID)

	if err != nil {
		return &skillResp, errors.New("skill does not exists")
	}
	parts := strings.Split(skill.ID, "-")
	lastPart := parts[len(parts)-1]

	skillResp = models.AgentSkillResponse{
		SkillId:          skill.ID,
		Name:             skill.Name,
		Description:      skill.Description,
		Type:             skill.Type,
		IsActive:         skill.IsActive,
		IsConfigured:     skill.IsConfigured,
		Avatar:           skill.Avatar,
		Config:           skill.Config,
		Tags:             skill.Tags,
		Category:         skill.Category,
		ShortDescription: skill.ShortDescription,
		LongDescription:  skill.LongDescription,
		SkillSlug:        fmt.Sprintf("%s-%s", slug.Make(skill.Name), lastPart),
	}

	return &skillResp, nil
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
