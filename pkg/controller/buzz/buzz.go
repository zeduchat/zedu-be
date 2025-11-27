package buzz

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
}

func (base *Controller) Create(c *gin.Context) {
	var req models.CreateBuzzRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing buzz request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for buzz request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := buzz.CreateBuzz(base.Db, base.Logger, req, userID.(string))
	if err != nil {
		base.Logger.Error("failed to create buzz: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("buzz created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "buzz created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) Join(c *gin.Context) {
	buzzID := c.Param("id")

	valid := utility.IsValidUUID(buzzID)
	if !valid {
		base.Logger.Info("invalid buzz ID format")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz ID format", "failed to decode buzz ID", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	resp, code, err := buzz.JoinBuzz(base.Db, base.Logger, buzzID, userID.(string))
	if err != nil {
		base.Logger.Error("failed to join buzz: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("user joined buzz successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "user joined buzz successfully", resp)
	c.JSON(http.StatusOK, rd)
}

// GetAgoraToken generates an Agora RTC token for joining a buzz
func (base *Controller) GetAgoraToken(c *gin.Context) {
	var req models.BuzzAgoraTokenRequest

	userID, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Info("error parsing Agora token request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		base.Logger.Info("validation failed for Agora token request")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	resp, code, err := agora.GetAgoraToken(base.Db, base.Logger, req.BuzzID, userID.(string))
	if err != nil {
		base.Logger.Error("failed to generate Agora token: %v", err)
		rd := utility.BuildErrorResponse(code, "error", err.Error(), err, nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("Agora token generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agora token generated successfully", resp)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) LeaveBuzz(c *gin.Context) {
	buzzID, ok := c.Params.Get("id")
	if !ok || !(utility.IsValidUUID(buzzID)) {
		base.Logger.Error("invalid request param: buzz id is invalid")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid buzz id in params", errors.New("invalid buzz id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userIDInterface, err := middleware.GetUserClaims(c, base.Db.Postgresql, "user_id")
	if err != nil {
		base.Logger.Info("unable to fetch user claims")
		rd := utility.BuildErrorResponse(http.StatusUnauthorized, "error", "authentication required", err, nil)
		c.JSON(http.StatusUnauthorized, rd)
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		base.Logger.Error("user_id is not of type string")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "user_id is not of type string", errors.New("user_id is not of type string"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	data, statusCode, err := buzz.LeaveBuzz(base.Db, base.Logger, buzzID, userID)

	if err != nil {
		base.Logger.Error("Failed to leave buzz: %v", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("user %s left buzz %s successfully", userID, buzzID)
	rd := utility.BuildSuccessResponse(statusCode, "user left buzz successfully", data)
	c.JSON(statusCode, rd)
}
