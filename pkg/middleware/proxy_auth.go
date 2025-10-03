package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func ProxyKeyAuthMiddleware(db *gorm.DB, logger *utility.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Error("Missing or invalid authorization header")
			rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Missing proxy key", "Unauthorized", nil)
			c.JSON(http.StatusUnauthorized, rd)
			c.Abort()
			return
		}

		clientKey := strings.TrimPrefix(authHeader, "Bearer ")
		clientKey = strings.TrimSpace(clientKey)

		orgID, agentID, err := models.ValidateAgentApiKey(db, clientKey)
		if err != nil {
			logger.Error("Invalid proxy key provided", err)
			rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "Invalid proxy key", "Unauthorized", nil)
			c.JSON(http.StatusUnauthorized, rd)
			c.Abort()
			return
		}

		c.Set("org_id", orgID)
		c.Set("agent_id", agentID)
		c.Set("preshared_key", clientKey)
		c.Next()
	}
}
