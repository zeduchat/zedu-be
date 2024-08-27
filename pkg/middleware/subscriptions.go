package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func MonitorFreeSub(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		owner_id, _ := GetIdFromToken(c)

		user, err := user.GetUserByID(db, owner_id)
		if err != nil {
			r := utility.BuildErrorResponse(http.StatusNotFound, "error", "User not Found", "Unauthorized", nil)
			c.AbortWithStatusJSON(http.StatusNotFound, r)
			return
		}

		if user.SubscriptionPlanId == "free" {

			daysSinceCreated := time.Since(user.CreatedAt).Hours() / 24

			if daysSinceCreated >= 30 {
				r := utility.BuildErrorResponse(http.StatusForbidden, "error", "Subscription expired", "Please upgrade your plan", nil)
				c.AbortWithStatusJSON(http.StatusForbidden, r)
				return
			}
		}

		c.Next()
	}
}
