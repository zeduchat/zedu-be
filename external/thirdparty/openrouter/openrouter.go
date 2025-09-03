package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/config"
)

func (r *RequestObj) GetChatCompletions() (map[string]any, error) {
	var (
		openRouterResponse map[string]any
		logger             = r.Logger
		reqData            = r.RequestData
		config             = config.GetConfig()
	)
	req, ok := reqData.(external_models.OpenRouterReq)
	if !ok {
		logger.Error("open router get chat completions", reqData, "request data format error")
		return openRouterResponse, fmt.Errorf("request data format error")
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		logger.Error("failed to marshal request", err)
		return openRouterResponse, err
	}

	request := bytes.NewBuffer(reqBody)
	apiKey := config.App.OpenRouterApiKey
	if apiKey == "" {
		logger.Error("open router get chat completions", "detail:", "config apiKey not found")
		return openRouterResponse, fmt.Errorf("config apiKey not found")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
	err = r.getNewSendRequestObject(request, headers, "/chat/completions").SendRequest(&openRouterResponse)
	if err != nil {
		logger.Error("open router get chat completions", openRouterResponse, err.Error())
		return openRouterResponse, err
	}
	return openRouterResponse, nil
}

func (r *RequestObj) GetStreamChatCompletions(ctx context.Context) (<-chan external_models.StreamChunk, error) {
	var (
		openRouterResponse <-chan external_models.StreamChunk
		logger             = r.Logger
		reqData            = r.RequestData
		config             = config.GetConfig()
	)
	req, ok := reqData.(external_models.OpenRouterReq)
	if !ok {
		logger.Error("open router get chat completions", reqData, "request data format error")
		return openRouterResponse, fmt.Errorf("request data format error")
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		logger.Error("failed to marshal request", err)
		return openRouterResponse, err
	}

	logger.Info("OpenRouter request body: %s", string(reqBody)) 

	request := bytes.NewBuffer(reqBody)
	apiKey := config.App.OpenRouterApiKey
	if apiKey == "" {
		logger.Error("open router get chat completions", "detail:", "config apiKey not found")
		return openRouterResponse, fmt.Errorf("config apiKey not found")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",  
		"Accept":        "text/event-stream", 
		"Cache-Control": "no-cache",
	}

	stream, err := r.getNewStreamingObject(request, headers, "/chat/completions", ctx).SendStream()
	if err != nil {
		logger.Error("open router get chat completions", openRouterResponse, err.Error())
		return openRouterResponse, err
	}

	return stream, nil
}

func (r *RequestObj) GetAllModels() (external_models.OpenRouterModelsResponse, error) {
	var (
		openRouterModelsResponse external_models.OpenRouterModelsResponse
		logger                   = r.Logger
		config                   = config.GetConfig()
		reqData                  = r.RequestData
	)

	apiKey := config.App.OpenRouterApiKey
	if apiKey == "" {
		logger.Error("open router get all models", "detail:", "config apiKey not found")
		return openRouterModelsResponse, fmt.Errorf("config apiKey not found")
	}

	headers := map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}

	fetchTools := reqData.(bool)
	if fetchTools {
		err := r.getNewSendRequestObject(nil, headers, "/models?supported_parameters=tools").SendRequest(&openRouterModelsResponse)
		if err != nil {
			logger.Error("open router get all models", openRouterModelsResponse, err.Error())
			return openRouterModelsResponse, err
		}
	}

	err := r.getNewSendRequestObject(nil, headers, "/models").SendRequest(&openRouterModelsResponse)
	if err != nil {
		logger.Error("open router get all models", openRouterModelsResponse, err.Error())
		return openRouterModelsResponse, err
	}
	return openRouterModelsResponse, nil
}
