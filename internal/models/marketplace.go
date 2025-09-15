package models

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

// Agents

func (i *Integrations) GetFeaturedAgents(db *gorm.DB, c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error, int) {
	var agents []Integrations
	pagination := postgresql.GetPagination(c)

	query := db.Model(&Integrations{}).
		Where("is_active = ?", true).
		Order("stars DESC").
		Limit(10)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"stars",
		"desc",
		pagination,
		&agents,
		nil,
	)
	if err != nil {
		return agents, paginationResponse, err, http.StatusInternalServerError
	}
	return agents, paginationResponse, nil, http.StatusOK
}

func (i *Integrations) GetPopularAgents(db *gorm.DB, c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error, int) {
	var agents []Integrations
	pagination := postgresql.GetPagination(c)

	// Join organisation_integrations and count installs
	query := db.Model(&Integrations{}).
		Select("integrations.*, COUNT(organisation_integrations.id) AS install_count").
		Joins("LEFT JOIN organisation_integrations ON organisation_integrations.integration_id = integrations.id").
		// Where("integrations.is_active = ?", true).
		Group("integrations.id").
		Order("install_count DESC").
		Limit(10)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"install_count",
		"desc",
		pagination,
		&agents,
		nil,
	)
	if err != nil {
		return agents, paginationResponse, err, http.StatusInternalServerError
	}
	return agents, paginationResponse, nil, http.StatusOK
}

func GetUniqueCategories(db *gorm.DB) ([]string, error) {
	var categories []string
	if err := db.
		Model(&Integrations{}).
		Where("category IS NOT NULL AND category != ''").
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (i *Integrations) GetAgentsByCategory(db *gorm.DB, c *gin.Context, categories []string, sortBy string) ([]Integrations, postgresql.PaginationResponse, error, int) {
	var agents []Integrations
	pagination := postgresql.GetPagination(c)

	query := db.Model(&Integrations{}).
		Where("category IN ?", categories)

	sortOrder := "created_at"
	sortBy, ok := map[string]string{"name": "name", "rating": "stars"}[sortBy]
	if ok {
		sortOrder = sortBy
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortOrder,
		"desc",
		pagination,
		&agents,
		nil,
	)
	if err != nil {
		return agents, paginationResponse, err, http.StatusInternalServerError
	}
	return agents, paginationResponse, nil, http.StatusOK
}

// needs more check
func (i *Integrations) FilterAgents(db *gorm.DB, filters map[string]interface{}, c *gin.Context) ([]Integrations, postgresql.PaginationResponse, error, int) { //
	var agents []Integrations
	pagination := postgresql.GetPagination(c)

	query := db.Model(&Integrations{})
	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&agents,
		nil,
	)
	if err != nil {
		return agents, paginationResponse, err, http.StatusInternalServerError
	}

	return agents, paginationResponse, nil, http.StatusOK
}

func (i *Integrations) SearchAgents(db *gorm.DB, c *gin.Context, keyword, sortBy string) ([]Integrations, postgresql.PaginationResponse, error, int) {
	var agents []Integrations
	pagination := postgresql.GetPagination(c)
	searchTerm := "%" + keyword + "%"

	query := db.Model(&Integrations{}).
		Where("name ILIKE ? OR title ILIKE ? OR app_description ILIKE ?", searchTerm, searchTerm, searchTerm)

	sortOrder := "created_at"
	sortBy, ok := map[string]string{"name": "name", "rating": "stars"}[sortBy]
	if ok {
		sortOrder = sortBy
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortOrder,
		"desc",
		pagination,
		&agents,
		nil,
	)
	if err != nil {
		return agents, paginationResponse, err, http.StatusInternalServerError
	}

	return agents, paginationResponse, nil, http.StatusOK
}

// skills

func (s *GeneralAgentSkill) GetSkillsByCategory(db *gorm.DB, c *gin.Context, categories []string, sortBy string) ([]GeneralAgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []GeneralAgentSkill
	pagination := postgresql.GetPagination(c)

	query := db.Model(&GeneralAgentSkill{}).
		Where("category IN ?", categories, true)

	sortOrder := "created_at"
	sortBy, ok := map[string]string{"name": "name", "rating": "stars"}[sortBy]
	if ok {
		sortOrder = sortBy
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortOrder,
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

func (s *GeneralAgentSkill) SearchSkills(db *gorm.DB, c *gin.Context, keyword, sortBy string) ([]GeneralAgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []GeneralAgentSkill
	pagination := postgresql.GetPagination(c)
	searchTerm := "%" + keyword + "%"

	query := db.Model(&GeneralAgentSkill{}).
		Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)

	sortOrder := "created_at"
	sortBy, ok := map[string]string{"name": "name", "rating": "stars"}[sortBy]
	if ok {
		sortOrder = sortBy
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortOrder,
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

func GetUniqueSkillsCategories(db *gorm.DB) ([]string, error) {
	var categories []string

	if err := db.Model(&GeneralAgentSkill{}).
		Where("category IS NOT NULL AND category != ''").
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

// workflow

func (w *GeneralWorkflow) GetWorkflowsByCategory(db *gorm.DB, c *gin.Context, categories []string, sortBy string) (*[]GeneralWorkflow, postgresql.PaginationResponse, error, int) {
	var workflows []GeneralWorkflow
	pagination := postgresql.GetPagination(c)

	query := db.Model(&GeneralWorkflow{}).
		Where("category IN ?", categories)
	sortOrder := "created_at"
	sortBy, ok := map[string]string{"name": "name", "rating": "stars"}[sortBy]
	if ok {
		sortOrder = sortBy
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortOrder,
		"desc",
		pagination,
		&workflows,
		nil,
	)
	if err != nil {
		return &workflows, paginationResponse, err, http.StatusInternalServerError
	}
	return &workflows, paginationResponse, nil, http.StatusOK
}

func (w *GeneralWorkflow) SearchWorkflows(db *gorm.DB, c *gin.Context, keyword, sortBy string) (*[]GeneralWorkflow, postgresql.PaginationResponse, error, int) {
	var workflows []GeneralWorkflow
	pagination := postgresql.GetPagination(c)
	searchTerm := "%" + keyword + "%"

	query := db.Model(&GeneralWorkflow{}).
		Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
	sortOrder := "created_at"
	sortBy, ok := map[string]string{"name": "name", "rating": "stars"}[sortBy]
	if ok {
		sortOrder = sortBy
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		sortOrder,
		"desc",
		pagination,
		&workflows,
		nil,
	)
	if err != nil {
		return &workflows, paginationResponse, err, http.StatusInternalServerError
	}
	return &workflows, paginationResponse, nil, http.StatusOK
}

func GetUniqueWorkflowCategories(db *gorm.DB) ([]string, error) {
	var categories []string
	if err := db.
		Model(&GeneralWorkflow{}).
		Where("category IS NOT NULL AND category != ''").
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}
