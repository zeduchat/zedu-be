package telexai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	aisvc "github.com/hngprojects/telex_be/services/telexai"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (tctrl *Controller) RespondToChat(c *gin.Context) {
	var req models.TelexAIChatCompletionsReq

	err := c.ShouldBindJSON(&req)
	if err != nil {
		tctrl.Logger.Info("error parsing request body")
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to parse request body", err, nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	err = tctrl.Validator.Struct(&req)
	if err != nil {
		tctrl.Logger.Info("validation failed")
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error",
			"Validation failed", utility.ValidationResponse(err, tctrl.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	response, code, err := aisvc.ChatCompletions(tctrl.Db, tctrl.Logger, req, tctrl.ExtReq)
	if err != nil {
		tctrl.Logger.Error("Failed to make chat completions", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to make chat completions", err.Error(), nil)
		c.JSON(code, rd)
		return
	}
	tctrl.Logger.Info("chat completed  successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "chat completed successfully", response)
	c.JSON(http.StatusOK, rd)
}
