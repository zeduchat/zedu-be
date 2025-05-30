package telexai

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

var (
	model = "google/gemma-3-4b-it:free"

	ErrNoOpenRouterAPIKeyConfig = errors.New("no open router api key configuration")
)

func ChatCompletions(db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, extReq request.ExternalRequest) (models.TelexAIChatCompletionsResp, int, error) {
	// TODO: OrgID is useful for handling payments and premium access per organisation
	openRouterPayload := external_models.OpenRouterReq{
		Model:    model,
		Messages: req.Messages,
	}
	res, err := extReq.SendExternalRequest(request.GetChatCompletions, openRouterPayload)
	if err != nil {
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}
	result, ok := res.(external_models.OpenRouterResp)
	if !ok {
		logger.Error("failed to get chat completions: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}
	if len(result.Choices) == 0 {
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, fmt.Errorf("empty response")
	}
	resp := models.TelexAIChatCompletionsResp{
		Messages: result.Choices[0].Message,
	}
	return resp, http.StatusOK, nil
}
