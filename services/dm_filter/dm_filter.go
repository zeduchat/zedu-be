package dmfilter

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
)

func FilterData(db *storage.Database, userId, orgId string, c *gin.Context) ([]models.DmFilter, *elastic.PaginationResponse, int, error) {
	return models.FilterDms(db, userId, orgId, c)
}
