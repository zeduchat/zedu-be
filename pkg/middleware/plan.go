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
		var currentPlan models.Plan

		cachedPlan, err := rd.RedisGet(rdb, cacheKey)
		if err == redis.Nil || len(cachedPlan) == 0 {

			orgPlan := &models.OrganisationPlan{}
			currentPlan, err = orgPlan.GetPlanByOrgID(db, orgID)
			if err != nil {
				r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", err.Error(), nil)
				c.AbortWithStatusJSON(http.StatusInternalServerError, r)
				return
			}

			if err := rd.RedisSetPerm(rdb, cacheKey, currentPlan); err != nil {
				r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to cache the plan", nil)
				c.AbortWithStatusJSON(http.StatusInternalServerError, r)
				return
			}
		} else {

			if err := json.Unmarshal(cachedPlan, &currentPlan); err != nil {
				r := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Internal Server Error", "Failed to unmarshal cached plan", nil)
				c.AbortWithStatusJSON(http.StatusInternalServerError, r)
				return
			}
		}

		c.Set("currentPlan", currentPlan)
		c.Next()
	}
}

func GetCurrentPlan(c *gin.Context) (models.Plan, error) {
	plan, exists := c.Get("currentPlan")
	if !exists {
		return models.Plan{}, errors.New("plan not found")
	}

	return plan.(models.Plan), nil
}
