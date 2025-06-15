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

	model, err := telexai.ExtractModel(c, base.Logger, req, base.ExtReq, base.Db.Redis)
	if err != nil {
		base.Logger.Error("failed to extract model", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to extract model", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	req.Model = model

	if req.Stream {
		//TODO: implement streaming later

		rd := utility.BuildErrorResponse(http.StatusNotImplemented, "error", "Streaming is not implemented yet", "Streaming is not implemented yet", nil)
		c.JSON(http.StatusNotImplemented, rd)
		return
	} else {
		response, code, err := telexai.ChatCompletions(base.Db, base.Logger, req, base.ExtReq, ids)
		if err != nil {
			base.Logger.Error("Failed to make chat completions", err)
			rd := utility.BuildErrorResponse(code, "error", "Failed to make chat completions", err.Error(), nil)
			c.JSON(code, rd)
			return
		}

		// perfom credit charge
		inputputLength := len(response.Messages.Content)
		var agentPrice float64 = 0.0 // temp value

		creditUsed := models.CalculateCreditCost(inputputLength, agentPrice)

		credit_usage := models.CreditUsage{
			ID:             utility.GenerateUUID(),
			OrganisationID: ids.OrganisationID,
			Amount:         creditUsed,
			AgentID:        ids.AgentID,
		}

		err = credit_usage.CreateCreditUsage(base.Db.Postgresql)
		if err != nil {
			base.Logger.Error("failed to create credit usage!!")
			rd := utility.BuildErrorResponse(400, "error", "failed to create organisation credit usage", err.Error(), nil)
			c.JSON(400, rd)
			return
		}

		if err = models.UpdateOrgCreditBalance(base.Db.Postgresql, ids.OrganisationID); err != nil {
			base.Logger.Error("Organisation credit Recalculation failed")
			rd := utility.BuildErrorResponse(400, "error", "organisation credit recalculation failed", err.Error(), nil)
			c.JSON(400, rd)
			return
		}

		base.Logger.Info("chat completed successfully")
		rd := utility.BuildSuccessResponse(http.StatusOK, "chat completed successfully", response)
		c.JSON(http.StatusOK, rd)
	}
}

func (base *Controller) ListAllModels(c *gin.Context) {
	models, err := telexai.ListAllModels(base.Logger, base.ExtReq, base.Db.Redis)
	if err != nil {
		base.Logger.Error("Failed to list all models", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to list all models", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("all models listed successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "all models listed successfully", models)
	c.JSON(http.StatusOK, rd)
}
