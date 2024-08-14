package helpcenter

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func CreateHelpCenterCategory(req models.CreateHelpCenterCategory , db *gorm.DB) (models.HelpCenterCategory, error) {
	helpCnt := models.HelpCenterCategory{
		ID:          		    utility.GenerateUUID(),
		Name:       		    req.Name,	
		Description:       		req.Description,
	}

	if err := helpCnt.CreateHelpCenterCategory(db, req.Name);

	err != nil {
		return models.HelpCenterCategory{}, err
	}

	return helpCnt, nil
}

func GetPaginatedCategories(c *gin.Context, db *gorm.DB) ([]models.HelpCntCategorySummary , postgresql.PaginationResponse, error) {
	helpCnt := models.HelpCenterCategory{}
	category, paginationResponse, err := helpCnt.GetAllCategories(db, c)

	if err != nil {
		return nil, paginationResponse, err
	}

	if len(category) == 0 {
		return []models.HelpCntCategorySummary{}, paginationResponse, nil
	}
	
	var topicSummaries []models.HelpCntCategorySummary
	for _, Hlp := range category {
		summary := models.HelpCntCategorySummary{
			ID: 		 	 Hlp.ID,
			Name:       	 Hlp.Name ,
			Description:     Hlp.Description,
			ArticlesLen:     len(Hlp.Articles),
		}
		topicSummaries = append(topicSummaries, summary)
	}

	return topicSummaries, paginationResponse, nil
}

func GetCategoryByID(db *gorm.DB, id string) (models.HelpCenterCategory, error) {// remove this soon
	helpCnt := models.HelpCenterCategory{}

	helpCnt.ID = id
	err := helpCnt.GetCategoryByID(db)
	if err != nil {
		return models.HelpCenterCategory{}, err
	}

	return helpCnt, nil
}

func GetArticlesByCategoryID(c *gin.Context, db *gorm.DB, id string) ([]models.HelpCenterArticle, postgresql.PaginationResponse, error) {
	helpCnt := models.HelpCenterCategory{ID: id}

	articles, paginationResponse, err := helpCnt.GetArticlesByCategoryID(db, c)

	if err != nil {
		return nil, paginationResponse, err
	}

	if len(articles) == 0 {
		return []models.HelpCenterArticle{}, paginationResponse, nil
	}	

	return articles, paginationResponse, nil
}
// func GetPaginatedArticles(c *gin.Context, db *gorm.DB) ([]models.HelpCenterArticle, postgresql.PaginationResponse, error) {
// 	helpCnt := models.HelpCenterArticle{}
// 	topics, paginationResponse, err := helpCnt.GetAllArticles(db, c)

// 	if err != nil {
// 		return nil, paginationResponse, err
// 	}

// 	if len(topics) == 0 {
// 		return []models.HelpCenterArticle{}, paginationResponse, nil
// 	}
	
// 	var topicSummaries []models.HelpCenterArticle
// 	for _, Hlp := range topics {
// 		summary := models.HelpCenterArticle{
// 			ID: 		 Hlp.ID,
// 			Title:       Hlp.Title,
// 		}
// 		topicSummaries = append(topicSummaries, summary)
// 	}

// 	return topicSummaries, paginationResponse, nil
// }