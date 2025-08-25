package token

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
)

func GetConnToken(userId string, db *gorm.DB) (gin.H, int, error) {

	userClaims := jwt.MapClaims{}

	userClaims["sub"] = userId
	userClaims["exp"] = time.Now().Unix() + int64(3600)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	connToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

	res := gin.H{
		"token": connToken,
	}

	return res, http.StatusOK, nil

}

func GetSubToken(userId string, req models.ChannelSubTokenReq, db *gorm.DB) (gin.H, int, error) {

	var (
		channelName = req.Channel
		org         = models.Organisation{}
		orgId       string
		cUserId     string
	)

	parts := strings.Split(req.Channel, "/")
	if len(parts) == 2 {
		orgId = parts[0]
		cUserId = parts[1]

		if _, err := uuid.Parse(orgId); err != nil {
			return gin.H{}, http.StatusBadRequest, errors.New("invalid channel id format")
		}

		exist, _ := org.CheckUserIsMemberOfOrg(userId, orgId, db)

		if !exist || cUserId != userId {
			return gin.H{}, http.StatusBadRequest, errors.New("invalid channel subscription")
		}

	} else {
		if _, err := uuid.Parse(channelName); err != nil {
			return gin.H{}, http.StatusBadRequest, errors.New("invalid channel id supplied")
		}

	}

	userClaims := jwt.MapClaims{}
	userClaims["sub"] = userId
	userClaims["channel"] = channelName
	userClaims["exp"] = time.Now().Unix() + int64(3600)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	subToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

	res := gin.H{
		"token": subToken,
	}

	return res, http.StatusOK, nil
}
