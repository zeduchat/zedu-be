package fcmtokens

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	fcmtokens "github.com/hngprojects/telex_be/services/fcmTokens"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateFcmToken(c *gin.Context) {
	var req models.CreateFcmTokenRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	req.UserId = userId

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := fcmtokens.CreateFcmToken(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating fcmtoken for user", err)
		rd := utility.BuildErrorResponse(code, "error", "error creating fcmtoken for user", err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("fcmtoken created for user successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "fcmtoken created for user successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) CreateWnsToken(c *gin.Context) {
	var req models.CreateWnsTokenRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	req.UserId = userId

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := fcmtokens.CreateWnsConfig(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating WNS config for user", err)
		rd := utility.BuildErrorResponse(code, "error", "error creating WNS config for user", err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("WNS config created for user successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "WNS config created for user successfully", nil)
	c.JSON(code, rd)
}

func (base *Controller) CreateWebPushToken(c *gin.Context) {
	var req models.CreateWebPushTokenRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)
	req.UserId = userId

	err := c.ShouldBindJSON(&req)
	if err != nil {
		base.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		base.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	code, err := fcmtokens.CreateWebPushToken(req, base.Db.Postgresql)
	if err != nil {
		base.Logger.Error("error creating webpush config for user", err)
		rd := utility.BuildErrorResponse(code, "error", "error creating webpush config for user", err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("webpush config created for user successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "webpush config created for user successfully", nil)
	c.JSON(code, rd)
}
