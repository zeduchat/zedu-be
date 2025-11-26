package buzz

import (
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

// GetAgoraToken generates an Agora RTC token for joining a huddle
func (base *Controller) GetAgoraToken(c *gin.Context) {
	var req models.HuddleAgoraTokenRequest

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

	resp, code, err := agora.GetAgoraToken(base.Db, base.Logger, req.HuddleID, userID.(string))
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
