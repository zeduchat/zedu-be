package translator

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/translator"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) CreatePrompt(c *gin.Context) {
	var req models.Prompts

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

	statusCode, err := translator.CreatePrompt(base.Db.Postgresql, req)
	if err != nil {
		base.Logger.Info("Error creating prompt")
		rd := utility.BuildErrorResponse(statusCode, "error", "Error creating prompt", err, err.Error())
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Prompt created successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Prompt created successfully", nil)
	c.JSON(statusCode, rd)
}

func (base *Controller) GetPrompts(c *gin.Context) {

	response, statusCode, err := translator.GetPrompts(base.Db.Postgresql)
	if err != nil {
		base.Logger.Info("Error fetching prompts")
		rd := utility.BuildErrorResponse(statusCode, "error", "Error fetching prompts", err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Prompts fetched successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Prompts fetched successfully", response)
	c.JSON(statusCode, rd)
}

func (base *Controller) GetPrompt(c *gin.Context) {
	prompt_id := c.Param("prompt_id")

	if _, err := uuid.Parse(prompt_id); err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid prompt id format", "", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	response, statusCode, err := translator.GetPrompt(base.Db.Postgresql, prompt_id)
	if err != nil {
		base.Logger.Info("Error fetching prompts")
		rd := utility.BuildErrorResponse(statusCode, "error", "Error fetching prompts", err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Prompts fetched successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Prompts fetched successfully", response)
	c.JSON(statusCode, rd)
}

func (base *Controller) FetchUniqueSteps(c *gin.Context) {
	response, statusCode, err := translator.FetchUniqueSteps(base.Db.Postgresql)
	if err != nil {
		base.Logger.Info("Error fetching unqiue steps")
		rd := utility.BuildErrorResponse(statusCode, "error", "Error fetching unqiue steps", err, nil)
		c.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("Unique Steps fetched successfully")
	rd := utility.BuildSuccessResponse(statusCode, "Unique Steps fetched successfully", response)
	c.JSON(statusCode, rd)
}
