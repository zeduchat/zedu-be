package models

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type HelpCenterArticle struct {
	ID         string             `gorm:"column:article_id; type:uuid; primaryKey" json:"article_id"`
	Title      string             `gorm:"column:title; type:varchar(255); not null" json:"title"`
	Content    string             `gorm:"column:content; type:varchar(255); not null" json:"content"`
	CategoryID string             `gorm:"column:category_id; type:uuid; not null" json:"category_id"`
	Category   HelpCenterCategory `gorm:"foreignKey:CategoryID" json:"-"`
	CreatedAt  time.Time          `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time          `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type HelpCenterCategory struct {
	ID          string              `gorm:"column:category_id; type:uuid; primaryKey" json:"category_id"`
	Name        string              `gorm:"column:name; type:varchar(255); not null" json:"name"`
	Articles    []HelpCenterArticle `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"articles"`
	Description string              `gorm:"column:description; type:varchar(255); not null" json:"description"`
	CreatedAt   time.Time           `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time           `gorm:"column:updated_at; null; autoUpdateTime" json:"updated_at"`
}

type CreateHelpCenterCategory struct {
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Description string `json:"description" validate:"required,min=2,max=255"`
}

type CreateHelpCenterArticle struct {
	Title      string `json:"title" validate:"required,min=2,max=255"`
	Content    string `json:"content" validate:"required,min=2,max=255"`
	CategoryID string `json:"category_id" validate:"required"`
}

type HelpCntArticleSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}
type HelpCntCategorySummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ArticlesLen int    `json:"articles_len"`
}

func (j *HelpCenterCategory) CreateHelpCenterCategory(db *gorm.DB, name string) error {
	var hnpCntCategory HelpCenterCategory

	exists := postgresql.CheckExists(db, &hnpCntCategory, "name = ?", name)

	if exists {
		return errors.New("help center category already exists")
	}

	err := postgresql.CreateOneRecord(db, &j)

	if err != nil {
		return err
	}

	return nil
}

func (j *HelpCenterCategory) GetAllCategories(db *gorm.DB, c *gin.Context) ([]HelpCenterCategory, postgresql.PaginationResponse, error) {
	var helpCntCat []HelpCenterCategory

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db.Preload("Articles"),
		"created_at",
		"desc",
		pagination,
		&helpCntCat,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return helpCntCat, paginationResponse, nil
}

func (j *HelpCenterCategory) GetCategoryByID(db *gorm.DB) error {
	err := postgresql.SelectFirstFromDb(db.Preload("Articles"), &j)

	if err != nil {
		return err
	}

	return nil
}

func (j *HelpCenterCategory) GetArticlesByCategoryID(db *gorm.DB, c *gin.Context) ([]HelpCenterArticle, postgresql.PaginationResponse, error) {
	var articles []HelpCenterArticle

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&articles,
		"category_id = ?",
		j.ID,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return articles, paginationResponse, nil
}

func (j *HelpCenterArticle) CreateHelpCenterArticle(db *gorm.DB, title string) error {
	var hnpCntArticle HelpCenterArticle

	exists := postgresql.CheckExists(db, &hnpCntArticle, "title = ?", title)

	if exists {
		return errors.New("help center article already exists")
	}
	err := postgresql.CreateOneRecord(db, &j)

	if err != nil {
		return err
	}

	return nil
}

func (j *HelpCenterArticle) GetAllArticles(db *gorm.DB, c *gin.Context) ([]HelpCenterArticle, postgresql.PaginationResponse, error) {
	var helpCntArticles []HelpCenterArticle

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&helpCntArticles,
		nil,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return helpCntArticles, paginationResponse, nil
}

func (j *HelpCenterArticle) GetArticleByID(db *gorm.DB) error {
	err := postgresql.SelectFirstFromDb(db, &j)

	if err != nil {
		return err
	}

	return nil
}

func (h *HelpCenterArticle) SearchHelpCenterArticles(db *gorm.DB, c *gin.Context, query string) ([]HelpCenterArticle, postgresql.PaginationResponse, error) {
	var helpCntArticles []HelpCenterArticle

	pagination := postgresql.GetPagination(c)
	searchQuery := "%" + query + "%"
	whereClause := "title ILIKE ? OR content ILIKE ?"
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db.Preload("Category"),
		"created_at",
		"desc",
		pagination,
		&helpCntArticles,
		whereClause,
		searchQuery,
		searchQuery,
	)
	if err != nil {
		return nil, paginationResponse, err
	}

	return helpCntArticles, paginationResponse, nil
}

func (j *HelpCenterArticle) UpdateArticleByID(db *gorm.DB, ID string) (HelpCenterArticle, error) {
	j.ID = ID

	exists := postgresql.CheckExists(db, &HelpCenterArticle{}, "article_id = ?", ID)
	if !exists {
		return HelpCenterArticle{}, gorm.ErrRecordNotFound
	}

	_, err := postgresql.SaveAllFields(db, j)
	if err != nil {
		return HelpCenterArticle{}, err
	}

	updatedHelpCntArticle := HelpCenterArticle{}
	err = db.First(&updatedHelpCntArticle, "article_id = ?", ID).Error
	if err != nil {
		return HelpCenterArticle{}, err
	}

	return updatedHelpCntArticle, nil
}

func (j *HelpCenterArticle) DeleteArticleByID(db *gorm.DB, ID string) error {

	exists := postgresql.CheckExists(db, &HelpCenterArticle{}, "article_id = ?", ID)
	if !exists {
		return gorm.ErrRecordNotFound
	}

	err := postgresql.DeleteRecordFromDb(db, &j)

	if err != nil {
		return err
	}

	return nil
}
