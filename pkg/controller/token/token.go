package token

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/token"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GetConnToken(c *gin.Context) {

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	userId := userClaims["user_id"].(string)

	respData, code, err := token.GetConnToken(userId, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("token generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "token generated successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) GetSubToken(c *gin.Context) {
	var req models.ChannelSubTokenReq

	err := c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")

	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)

	userId := userClaims["user_id"].(string)

	respData, code, err := token.GetSubToken(userId, req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("token generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "token generated successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) GetConnTokenUnAuth(c *gin.Context) {

	SubToken := c.GetHeader("X-Notification-Token")

	if SubToken == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid token", errors.New("X-Notification-Token header missing"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	accessToken := models.AccessToken{}
	accessToken.SubAccessToken = SubToken

	if _, err := uuid.Parse(SubToken); err == nil {
		var userUnAuth models.UnauthenticatedUser
		claims, exists := c.Get("client_ip")
		if !exists {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		valid, err := userUnAuth.VerifySubtoken(base.Db.Postgresql, models.UnauthReq{UserIdentifier: claims.(string), Limit: models.CHATUSAGELIMIT})

		if !valid || err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid token", err.Error(), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		accessToken.OwnerID = "00000000-0000-0000-0000-000000000000"
	}

	code, err := accessToken.GetBySubToken(base.Db.Postgresql)

	if (err != nil || !accessToken.IsLive) && accessToken.OwnerID != "00000000-0000-0000-0000-000000000000" {
		rd := utility.BuildErrorResponse(code, "error", "invalid notification token", errors.New("invalid or exipired notification token"), nil)
		c.JSON(code, rd)
		return
	}

	respData, code, err := token.GetConnToken(accessToken.OwnerID, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("token generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "connection token generated successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) GetSubTokenUnAuth(c *gin.Context) {
	var req models.ChannelSubTokenReq

	SubToken := c.GetHeader("X-Notification-Token")

	if SubToken == "" {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid token", errors.New("X-Notification-Token header missing"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	accessToken := models.AccessToken{}
	accessToken.SubAccessToken = SubToken

	if _, err := uuid.Parse(SubToken); err == nil {
		var userUnAuth models.UnauthenticatedUser
		claims, exists := c.Get("client_ip")
		if !exists {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", errors.New("user not authorized"), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		valid, err := userUnAuth.VerifySubtoken(base.Db.Postgresql, models.UnauthReq{UserIdentifier: claims.(string), Limit: models.CHATUSAGELIMIT})

		if !valid || err != nil {
			rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid token", err.Error(), nil)
			c.JSON(http.StatusBadRequest, rd)
			return
		}

		accessToken.OwnerID = "00000000-0000-0000-0000-000000000000"
	}

	code, err := accessToken.GetBySubToken(base.Db.Postgresql)

	if (err != nil || !accessToken.IsLive) && accessToken.OwnerID != "00000000-0000-0000-0000-000000000000" {
		rd := utility.BuildErrorResponse(code, "error", "invalid notification token", errors.New("invalid or exipired notification token"), nil)
		c.JSON(code, rd)
		return
	}

	err = c.ShouldBindJSON(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	respData, code, err := token.GetSubToken(accessToken.OwnerID, req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("token generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "token generated successfully", respData)
	c.JSON(code, rd)
}
