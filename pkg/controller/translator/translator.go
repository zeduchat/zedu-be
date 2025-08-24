package translator

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/assets"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/translator"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) GenerateTranslation(c *gin.Context) {
	var req models.TranslationRequest

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

	response, statusCode, err := translator.GenerateTranslation(base.Db.Postgresql, base.Logger, base.ExtReq, req)
	if err != nil {
		base.Logger.Error("error generating translation", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Translation generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Translation generated successfully", response)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) GenerateWorkflowJSON(c *gin.Context) {
	agentID := c.Param("agent_id")
	if _, err := uuid.Parse(agentID); err != nil {
		base.Logger.Error("invalid agent id format", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Invalid agent id format", "failed to decode agent id", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, statusCode, err := translator.GenerateWorkflowJSON(base.Db.Postgresql, base.Logger, base.ExtReq, agentID)
	if err != nil {
		base.Logger.Error("error generating translation", err)
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Translation generated successfully")
	rd := utility.BuildSuccessResponse(http.StatusCreated, "Translation generated successfully", response.ProcessStep[len(response.ProcessStep)-1].Output)
	c.JSON(http.StatusCreated, rd)
}

func (base *Controller) TranslationTester(c *gin.Context) {
	htmlContent, err := assets.StaticFiles.ReadFile("static/tester.html")
	if err != nil {
		base.Logger.Error("Failed to read tester.html", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load tester page"})
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, string(htmlContent))
}
