package teams

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
	"github.com/hngprojects/telex_be/services/teams"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) CreateTeam(c *gin.Context) {
	var req models.CreateTeamRequest

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

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

	respData, err := teams.CreateTeam(base.Db.Postgresql, req, userId)
	if err != nil {
		base.Logger.Info("error creating team")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("team created successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "team created successfully", respData)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GetTeamByID(c *gin.Context) {
	teamID := c.Param("teamId")

	if _, err := uuid.Parse(teamID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid team id format", errors.New("failed to parse team id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, err := teams.GetTeamByID(base.Db.Postgresql, teamID)
	if err != nil {
		base.Logger.Info("error fetching team")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}


	base.Logger.Info("team fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "team fetched successfully", respData)
	c.JSON(http.StatusOK, rd)

}

func (base *Controller) GetAllRoomsInTeam(c *gin.Context) {
	teamID := c.Param("teamId")

	if _, err := uuid.Parse(teamID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid team id format", errors.New("failed to parse team id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	respData, additionalInfo ,err := teams.GetAllRoomsInTeam(base.Db.Postgresql, teamID)
	if err != nil {
		base.Logger.Info("error fetching channels")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response := gin.H{
		"rooms": respData,
		"additional_info": additionalInfo,
	}

	base.Logger.Info("channels fetched successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "rooms fetched successfully", response)
	c.JSON(http.StatusOK, rd)
}

func (base *Controller) DeleteTeam(c *gin.Context) {
	teamID := c.Param("teamId")

	if _, err := uuid.Parse(teamID); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid team id format", errors.New("failed to parse team id"), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	claims, exists := c.Get("userClaims")
	if !exists {
		base.Logger.Info("error getting claims")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "error getting claims", nil, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	err := teams.DeleteTeam(base.Db.Postgresql, userId, teamID)
	if err != nil {
		base.Logger.Info("error deleting channel")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	base.Logger.Info("team deleted successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "team deleted successfully", nil)
	c.JSON(http.StatusOK, rd)
}
