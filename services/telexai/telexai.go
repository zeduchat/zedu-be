package telexai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

var (
	openRouterUrl = "https://openrouter.ai/api/v1/chat/completions"
	model         = "google/gemma-3-4b-it:free"

	ErrNoOpenRouterAPIKeyConfig = errors.New("no open router api key configuration")
)

func ChatCompletions(db *storage.Database, logger *utility.Logger, req models.TelexAIChatCompletionsReq, apiKey string) (models.TelexAIChatCompletionsResp, int, error) {
	if apiKey == "" {
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, ErrNoOpenRouterAPIKeyConfig
	}

	openRouterPayload := models.OpenRouterReq{
		Model:    model,
		Messages: req.Messages,
	}

	requestBody, err := json.Marshal(openRouterPayload)
	if err != nil {
		logger.Error("Failed to marshal payload: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}

	// OrgID is useful for handling payments and premium access per organisation
	fmt.Println("The organisation ID:", req.OrgID)

	request, err := http.NewRequest("POST", openRouterUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Error("Failed to create request: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}

	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		logger.Error("Error sending request: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading response body: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}

	var result models.OpenRouterResp
	err = json.Unmarshal(body, &result)
	if err != nil {
		logger.Error("Error decoding response: ", err)
		return models.TelexAIChatCompletionsResp{}, http.StatusInternalServerError, err
	}

	res := models.TelexAIChatCompletionsResp{
		Messages: result.Choices[0].Message,
	}
	return res, http.StatusOK, nil
}
