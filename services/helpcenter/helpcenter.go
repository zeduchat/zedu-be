package helpcenter

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateHelpCenterArticle(req models.CreateHelpCenterArticle, db *gorm.DB) (models.HelpCenterArticle, error) {
	helpCnt := models.HelpCenterArticle{
		ID:         utility.GenerateUUID(),
		Title:      req.Title,
		Content:    req.Content,
		CategoryID: req.CategoryID,
	}

	if err := helpCnt.CreateHelpCenterArticle(db, req.Title); err != nil {
		return models.HelpCenterArticle{}, err
	}

	return helpCnt, nil
}

func GetArticleByID(db *gorm.DB, id string) (models.HelpCenterArticle, error) {
	helpCnt := models.HelpCenterArticle{}

	helpCnt.ID = id
	err := helpCnt.GetArticleByID(db)
	if err != nil {
		return models.HelpCenterArticle{}, err
	}

	return helpCnt, nil
}

func SearchHelpCenterArticles(c *gin.Context, db *gorm.DB, query string) ([]models.HelpCenterArticle, postgresql.PaginationResponse, error) {
	var helpCnt models.HelpCenterArticle
	topics, paginationResponse, err := helpCnt.SearchHelpCenterArticles(db, c, query)

	if err != nil {
		return nil, paginationResponse, err
	}

	if len(topics) == 0 {
		return []models.HelpCenterArticle{}, paginationResponse, nil
	}

	var topicSummaries []models.HelpCenterArticle
	for _, topic := range topics {
		summary := models.HelpCenterArticle{
			ID:         topic.ID,
			Title:      topic.Title,
			Content:    topic.Content,
			CategoryID: topic.CategoryID,
		}
		topicSummaries = append(topicSummaries, summary)
	}

	return topicSummaries, paginationResponse, nil
}

func UpdateArticleByID(db *gorm.DB, helpCnt models.HelpCenterArticle, ID string) (models.HelpCenterArticle, error) {
	updatedHelpCnt, err := helpCnt.UpdateArticleByID(db, ID)

	if err != nil {
		return models.HelpCenterArticle{}, err
	}

	return updatedHelpCnt, nil
}

func DeleteArticleByID(db *gorm.DB, ID string) error {
	helpCnt := models.HelpCenterArticle{ID: ID}

	err := helpCnt.DeleteArticleByID(db, ID)
	if err != nil {
		return err
	}

	return nil
}
