package telexai

import (
	"bytes"
	"github.com/hngprojects/telex_be/internal/config"
	"io"
	"net/http"
	"strings"

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

func (base *Controller) ProxyToOpenRouter(c *gin.Context) {
	config := config.GetConfig()

	path := c.Request.URL.Path
	path = strings.TrimPrefix(path, "/api/v1/telexai")
	openRouterURL := config.OpenRouter.BaseUrl + path

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		base.Logger.Error("Failed to read request body", err)
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "Failed to read request body", err.Error(), nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	req, err := http.NewRequest(c.Request.Method, openRouterURL, bytes.NewBuffer(body))
	if err != nil {
		base.Logger.Error("Failed to create request to OpenRouter", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to create request", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	for key, values := range c.Request.Header {
		lower := strings.ToLower(key)
		if lower == "authorization" {
			req.Header.Set("Authorization", "Bearer "+config.OpenRouter.ApiKey)
		} else if lower == "accept-encoding" {
			continue
		} else {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	req.URL.RawQuery = c.Request.URL.RawQuery

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		base.Logger.Error("Failed to make request to OpenRouter", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to make request to OpenRouter", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		base.Logger.Error("Failed to read response body from OpenRouter", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to read response", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}
}

func (base *Controller) RespondToChat(c *gin.Context) {
	var (
		req models.TelexAIChatCompletionsReq
		w   = c.Writer
	)

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
		err := telexai.StreamChatCompletions(c.Writer, base.Db, base.Logger, req, base.ExtReq, ids)
		if err != nil {
			base.Logger.Error("streaming failed", err)
			if !c.Writer.Written() {
				rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Streaming failed", err.Error(), nil)
				c.JSON(http.StatusInternalServerError, rd)
			}
		}
		return
	}

	response, code, err := telexai.RespondToChat(w, base.Db, base.Logger, req, base.ExtReq, ids)
	if err != nil {
		base.Logger.Error("failed to get chat completions", err)
		rd := utility.BuildErrorResponse(code, "error", "Failed to get chat completions", err.Error(), nil)
		c.JSON(code, rd)
		return
	}

	base.Logger.Info("chat completed successfully")
	rd := utility.BuildSuccessResponse(code, "chat completed successfully", response)
	c.JSON(code, rd)
}

func (base *Controller) ListModels(c *gin.Context) {

	models, err := telexai.ListModels(base.Logger, base.ExtReq, base.Db.Redis, false)
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

func (base *Controller) ListToolsModels(c *gin.Context) {

	models, err := telexai.ListModels(base.Logger, base.ExtReq, base.Db.Redis, true)
	if err != nil {
		base.Logger.Error("Failed to list all tools models", err)
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", "Failed to list all tools models", err.Error(), nil)
		c.JSON(http.StatusInternalServerError, rd)
		return
	}

	base.Logger.Info("all models listed successfully")
	rd := utility.BuildSuccessResponse(http.StatusOK, "all tools models listed successfully", models)
	c.JSON(http.StatusOK, rd)
}
