package agents

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreateAgentSkill(c *gin.Context) {
	var (
		req models.CreateAgentSkillRequest
	)

	org_id := c.Param("org_id")

	if _, err := uuid.Parse(org_id); err != nil {
		base.Logger.Error("invalid organisation id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id format", "failed to decode organisation id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		base.Logger.Error("Invalid request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	if err := base.Validator.Struct(&req); err != nil {
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error", "Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "unable to get user claims", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	resp, err := agents.CreateAgentSkill(org_id, req, base.Db.Postgresql, base.ExtReq, userId)
	if err != nil {
		base.Logger.Error("Failed to Create Custom Agent, invalid url:  "+req.JSONUrl, err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), "Failed to create agent", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("Custom agent created successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Agent created successfully", resp)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetAgentSkill(c *gin.Context) {
	// Implementation for getting an agent skill
}
func (base *Controller) UpdateAgentSkill(c *gin.Context) {
	// Implementation for updating an agent skill
}

func (base *Controller) DeleteAgentSkill(c *gin.Context) {
	// Implementation for deleting an agent skill
}
