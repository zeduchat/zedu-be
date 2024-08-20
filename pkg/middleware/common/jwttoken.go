package common

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func GetAllUserClaims(c *gin.Context, theClaim string) jwt.MapClaims {
	claims, exists := c.Get(theClaim)
	if !exists {
		return nil
	}

	userClaims := claims.(jwt.MapClaims)

	return userClaims

}
