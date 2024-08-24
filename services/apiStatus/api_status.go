package apistatus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func UpdateAPIStatus(db *gorm.DB, data []byte) error {
	var request models.StatusRequest
	err := json.Unmarshal(data, &request)
	if err != nil {
		return err
	}

	for _, item := range request.APIGroup.Item {
		for _, subItem := range item.Item {
			for _, response := range subItem.Response {
				var status string
				var details string
				if response.StatusCode >= 200 && response.StatusCode < 300 {
					status = "operational"
					details = "All test passed"
				} else if response.StatusCode >= 400 && response.StatusCode < 500 {
					status = "degraded"
					details = "High response time detected"
				} else if response.StatusCode >= 500 {
					status = "down"
					details = "API not responding (HTTP 503)"
				}

				apistatus := models.APIStatus{
					ID:             utility.GenerateUUID(),
					APIGroup:       fmt.Sprintf("%s API", item.Name),
					Status:         status,
					LastChecked:    time.Now().UTC(),
					ResponseTimeMs: "",
					Details:        details,
				}

				err := apistatus.Create(db)

				if err != nil {
					return err
				}
			}

		}
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
