package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

func CheckIsDeactivated(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		owner_id, _ := GetIdFromToken(c)
		user, err := user.GetUserByID(db, owner_id)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "User not found",
			})
			c.Abort()
			return
		}
		if user.Deactivated {
			c.JSON(400, gin.H{
				"error": "User is deactivated",
			})

			c.Next()
		}
	}
}
