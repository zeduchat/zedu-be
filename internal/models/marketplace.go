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

func GetUniqueCategories(db *gorm.DB) ([]string, error) {
	var categories []string
	if err := db.
		Model(&Integrations{}).
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (i *Integrations) GetAgentsByCategory(db *gorm.DB, c *gin.Context, category string) ([]Integrations, postgresql.PaginationResponse, error, int) {
	var agents []Integrations
	pagination := postgresql.GetPagination(c)

	query := db.Model(&Integrations{}).
		Where("category = ? AND is_active = ?", category, true)

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

func (i *Integrations) SearchAgents(db *gorm.DB, c *gin.Context, keyword string) ([]Integrations, postgresql.PaginationResponse, error, int) {
	var agents []Integrations
	pagination := postgresql.GetPagination(c)
	searchTerm := "%" + keyword + "%"

	query := db.Model(&Integrations{}).
		Where("name ILIKE ? OR title ILIKE ? OR app_description ILIKE ?", searchTerm, searchTerm, searchTerm)

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

// skills

func (s *GeneralAgentSkill) GetSkillsByCategory(db *gorm.DB, c *gin.Context, category string) ([]GeneralAgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []GeneralAgentSkill
	pagination := postgresql.GetPagination(c)

	query := db.Model(&GeneralAgentSkill{}).
		Where("category = ? AND is_active = ?", category, true)

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

func (s *GeneralAgentSkill) SearchSkills(db *gorm.DB, c *gin.Context, keyword string) ([]GeneralAgentSkill, postgresql.PaginationResponse, error, int) {
	var skills []GeneralAgentSkill
	pagination := postgresql.GetPagination(c)
	searchTerm := "%" + keyword + "%"

	query := db.Model(&GeneralAgentSkill{}).
		Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)

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

func GetUniqueSkillsCategories(db *gorm.DB) ([]string, error) {
	var categories []string

	if err := db.Model(&GeneralAgentSkill{}).Distinct("category").Pluck("category", &categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

// workflow

func (w *GeneralWorkflow) GetWorkflowsByCategory(db *gorm.DB, c *gin.Context, category string) ([]GeneralWorkflow, postgresql.PaginationResponse, error, int) {
	var workflows []GeneralWorkflow
	pagination := postgresql.GetPagination(c)

	query := db.Model(&GeneralWorkflow{}).
		Where("category = ?", category)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&workflows,
		nil,
	)
	if err != nil {
		return workflows, paginationResponse, err, http.StatusInternalServerError
	}
	return workflows, paginationResponse, nil, http.StatusOK
}

func (w *GeneralWorkflow) SearchWorkflows(db *gorm.DB, c *gin.Context, keyword string) ([]GeneralWorkflow, postgresql.PaginationResponse, error, int) {
	var workflows []GeneralWorkflow
	pagination := postgresql.GetPagination(c)
	searchTerm := "%" + keyword + "%"

	query := db.Model(&GeneralWorkflow{}).
		Where("name ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&workflows,
		nil,
	)
	if err != nil {
		return workflows, paginationResponse, err, http.StatusInternalServerError
	}
	return workflows, paginationResponse, nil, http.StatusOK
}

func GetUniqueWorkflowCategories(db *gorm.DB) ([]string, error) {
	var categories []string
	if err := db.
		Model(&GeneralWorkflow{}).
		Distinct("category").
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}
