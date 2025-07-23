package middleware

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

type TokenDetailDTO struct {
	AccessUuid  string `json:"access_uuid"`
	AccessToken string `json:"access_token"`
	ExpiresAt   time.Time
}

func CreateToken(user models.User, c *gin.Context) (*TokenDetailDTO, error) {

	var (
		tokenData = &TokenDetailDTO{}
		config    = config.GetConfig()
		err       error
	)

	tokenData.ExpiresAt = time.Now().AddDate(0, 0, config.Server.AccessTokenExpireDuration) // token valid for env set days
	tokenData.AccessUuid = user.ID
	tokenData.AccessUuid = utility.GenerateUUID()

	//create token
	userClaims := jwt.MapClaims{}

	// specify user claims
	userClaims["user_id"] = user.ID
	userClaims["access_uuid"] = tokenData.AccessUuid
	userClaims["role_id"] = user.OrgRoleID
	userClaims["org_id"] = user.CurrentOrg
	userClaims["exp"] = tokenData.ExpiresAt.Unix()
	userClaims["authorised"] = true

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	tokenData.AccessToken, err = token.SignedString([]byte(config.Server.Secret))
	if err != nil {
		return tokenData, err
	}

	c.Set("userRoleClaims", user.OrgRoleID)

	return tokenData, nil
}

// verify token
func verifyToken(tokenString string) (*jwt.Token, error) {
	config := config.GetConfig()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.Server.Secret), nil
	})
	if err != nil {
		fmt.Println("===================", err)
		return token, fmt.Errorf("Unauthorized")
	}
	return token, nil
}

// check if token is valid
func TokenValid(bearerToken string) (*jwt.Token, error) {
	token, err := verifyToken(bearerToken)
	if err != nil {
		if token != nil {
			return token, err
		}
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("Unauthorized")
	}

	return token, nil
}

func GetUserClaims(c *gin.Context, db *gorm.DB, theValue string) (any, error) {

	claims, exists := c.Get("userClaims")
	if !exists {
		return nil, errors.New("user claims not found")
	}

	userClaims := claims.(jwt.MapClaims)
	userValue, ok := userClaims[theValue]
	if !ok {
		return nil, errors.New("invalid value")
	}

	return userValue, nil

}

func CreateAdminToken(admin models.Admin, c *gin.Context) (*TokenDetailDTO, error) {

	var (
		tokenData = &TokenDetailDTO{}
		config    = config.GetConfig()
		err       error
	)

	tokenData.ExpiresAt = time.Now().AddDate(0, 0, config.Server.AccessTokenExpireDuration) // token valid for env set days
	tokenData.AccessUuid = admin.ID
	tokenData.AccessUuid = utility.GenerateUUID()

	//create token
	adminClaims := jwt.MapClaims{}

	// specify admin claims
	adminClaims["admin_id"] = admin.ID
	adminClaims["role"] = admin.Role
	adminClaims["access_uuid"] = tokenData.AccessUuid
	adminClaims["exp"] = tokenData.ExpiresAt.Unix()
	adminClaims["authorised"] = true

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, adminClaims)

	tokenData.AccessToken, err = token.SignedString([]byte(config.Server.Secret))
	if err != nil {
		return tokenData, err
	}

	c.Set("admin_id", admin.ID)

	return tokenData, nil
}

func GetAdminClaims(c *gin.Context, db *gorm.DB, theValue string) (any, error) {

	claims, exists := c.Get("adminClaims")
	if !exists {
		return nil, errors.New("admin claims not found")
	}

	adminClaims := claims.(jwt.MapClaims)
	adminValue, ok := adminClaims[theValue]
	if !ok {
		return nil, errors.New("invalid value")
	}

	return adminValue, nil

}
