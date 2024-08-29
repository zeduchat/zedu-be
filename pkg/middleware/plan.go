package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware/common"
	rd "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func PlanMiddleware(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userClaims := common.GetAllUserClaims(c)
		orgID, ok := userClaims["org_id"].(string)
		if !ok {
			r := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Unauthorized", "Missing org id", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, r)
			return
		}

		cacheKey := "org_plan:" + orgID
		var currentPlan models.OrganisationPlan

		cachedPlan, err := rd.RedisGet(rdb, cacheKey)
		if err == redis.Nil || len(cachedPlan) == 0 {

			currentPlan, err := currentPlan.GetAnOrgPlanById(db, orgID)
			if err != nil {
				r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to load plan", nil)
				c.AbortWithStatusJSON(http.StatusInternalServerError, r)
				return
			}

			planJSON, _ := json.Marshal(currentPlan)
			rd.RedisSet(rdb, cacheKey, planJSON)
		} else {
			if err := json.Unmarshal([]byte(cachedPlan), &currentPlan); err != nil {
				r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to unmarshal plan", nil)
				c.AbortWithStatusJSON(http.StatusInternalServerError, r)
				return
			}
		}

		c.Set("currentPlan", currentPlan)
		c.Next()
	}
}

func GetCurrentPlan(c *gin.Context) (models.OrganisationPlan, error) {
	plan, exists := c.Get("currentPlan")
	if !exists {
		return models.OrganisationPlan{}, errors.New("plan not found")
	}

	return plan.(models.OrganisationPlan), nil
}
