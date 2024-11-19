package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/auth"
	telexaudit "github.com/hngprojects/telex_be/services/telexAudit"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) RegisterUser(c *gin.Context) {
	var req models.CreateUserRequestModel

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

	reqData, err := auth.ValidateCreateUserRequest(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := auth.CreateUser(c, base.ExtReq, reqData, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, respData)
		base.Logger.Error("error saving user: ", err.Error())
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user created successfully")

	err = telexaudit.SignupAudit(base.Db, base.Logger, respData)
	if err != nil {
		base.Logger.Error("error broadcasting signup audit: ", err.Error())
	}

	rd := utility.BuildSuccessResponse(http.StatusCreated, "User Created Successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) CreateAdmin(c *gin.Context) {
	var req models.CreateUserRequestModel

	err := c.ShouldBind(&req)
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

	reqData, err := auth.ValidateCreateUserRequest(req, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := auth.CreateAdmin(reqData, base.Db.Postgresql, c)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "User Created Successfully", respData)
	c.JSON(code, rd)
}

func (base *Controller) LoginUser(c *gin.Context) {
	var req models.LoginRequestModel

	err := c.ShouldBind(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = base.Validator.Struct(&req)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := auth.LoginUser(req, base.Db.Postgresql, c, base.ExtReq)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user login successfully")

	err = telexaudit.LoginAudit(base.Db, base.Logger, respData)
	if err != nil {
		base.Logger.Error("error broadcasting login audit: ", err.Error())
	}

	rd := utility.BuildSuccessResponse(http.StatusOK, "user login successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) LogoutUser(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)

	access_uuid, _ := userClaims["access_uuid"].(string)
	owner_id, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get access id", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := auth.LogoutUser(access_uuid, owner_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user logout successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "user logout successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) GetOnboardStatus(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)

	owner_id, ok := userClaims["user_id"].(string)
	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get access id", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, code, err := auth.GetOnboardStatus(owner_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user status fetch successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "user status fetch successfully", respData)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) UpdateOnboardStatus(c *gin.Context) {

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	owner_id, ok := userClaims["user_id"].(string)

	if !ok {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get access id", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	respData, code, err := auth.UpdateOnboardStatus(owner_id, base.Db.Postgresql)
	if err != nil {
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("user onboarding status updated successfully")

	rd := utility.BuildSuccessResponse(http.StatusOK, "user onboarding status updated successfully", respData)
	c.JSON(http.StatusOK, rd)
}
