package telexai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/actions"
	"github.com/hngprojects/telex_be/services/telexai"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) RespondToChat(c *gin.Context) {
	var req models.TelexAIChatCompletionsReq

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
		rd := utility.BuildErrorResponse(http.StatusUnprocessableEntity, "error",
			"Validation failed", utility.ValidationResponse(err, base.Validator), nil)
		c.JSON(http.StatusUnprocessableEntity, rd)
		return
	}

	ids, err := actions.FetchAPIKeyCredentials(c)
	if err != nil {
		base.Logger.Error("failed to fetch api key credentials", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "failed to fetch api key credentials", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	//see if the user has enough credits to make the request
	// if !actions.HasEnoughCredits(base.Db, ids.UserID, req.Model, req.Messages) {
	// 	base.Logger.Error("user does not have enough credits to make the request")
	// 	rd := utility.BuildErrorResponse(http.StatusPaymentRequired, "error", "Insufficient credits to make the request", "Insufficient credits", nil)
	// 	c.JSON(http.StatusPaymentRequired, rd)
	// 	return
	// }

	model := telexai.ExtractModel(c, base.Logger, req)
	req.Model = model

	if req.Stream {
		//TODO: implement streaming later
	} else {
		response, code, err := telexai.ChatCompletions(base.Db, base.Logger, req, base.ExtReq, ids)
		if err != nil {
			base.Logger.Error("Failed to make chat completions", err)
			rd := utility.BuildErrorResponse(code, "error", "Failed to make chat completions", err.Error(), nil)
			c.JSON(code, rd)
			return
		}
		base.Logger.Info("chat completed successfully")
		rd := utility.BuildSuccessResponse(http.StatusOK, "chat completed successfully", response)
		c.JSON(http.StatusOK, rd)
	}
}

func (base *Controller) ListModels(c *gin.Context) {
	models, err := telexai.ListModels()
	if err != nil {
		base.Logger.Error("Failed to list models", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to list models", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("models listed successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "models listed successfully", models)
	c.JSON(http.StatusOK, rd)
}
