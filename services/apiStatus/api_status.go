package apistatus

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

func UpdateAPIStatus(db *gorm.DB, data []byte) error {
	var request models.StatusRequest
	err := json.Unmarshal(data, &request)
	if err != nil {
		return err
	}

	for _, data := range request.APIGroup.Item{
		
	}

	return nil
}

func GetAPIStatus(db *gorm.DB, c *gin.Context) ([]models.APIStatus, postgresql.PaginationResponse, error) {
	var (
		apiStatus models.APIStatus
	)

	apiStatuses, paginationResponse, err := apiStatus.GetAPIStatuses(db, c)
	if err != nil {
		return nil, paginationResponse, err
	}

	return apiStatuses, paginationResponse, nil
}
