package apistatus

import (
	"encoding/json"
	"fmt"
	"strconv"
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

	responseTimes := make(map[string][]int)

	for _, exec := range request.Run.Executions {
		if len(exec.Request.URL.Path) > 2 {
			apiGroup := exec.Request.URL.Path[2]

			if exec.Response.ResponseTime > 0 {
				responseTimes[apiGroup] = append(responseTimes[apiGroup], exec.Response.ResponseTime)
			}
		}
	}

	for apiGroup, times := range responseTimes {
		var totalResponseTime int
		for _, time := range times {
			totalResponseTime += time
		}

		averageResponseTime := totalResponseTime / len(times)

		var status, details string
		if averageResponseTime < 300 {
			status = "operational"
			details = "All tests passed"
		} else if averageResponseTime >= 300 && averageResponseTime < 500 {
			status = "degraded"
			details = "High response time detected"
		} else {
			status = "down"
			details = "API not responding (HTTP 503)"
		}

		apistatus := models.APIStatus{
			ID:             utility.GenerateUUID(),
			APIGroup:       fmt.Sprintf("%s API", apiGroup),
			Status:         status,
			LastChecked:    time.Now().UTC(),
			ResponseTimeMs: strconv.Itoa(averageResponseTime),
			Details:        details,
		}

		err := apistatus.Create(db)
		if err != nil {
			return err
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
