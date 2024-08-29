package common

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func GetAllUserClaims(c *gin.Context) jwt.MapClaims {
	claims, exists := c.Get("userClaims")
	if !exists {
		return nil
	}

	userClaims := claims.(jwt.MapClaims)

	return userClaims

}
