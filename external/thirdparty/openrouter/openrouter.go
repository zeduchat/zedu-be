package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/internal/config"
)

func (r *RequestObj) GetChatCompletions() (external_models.OpenRouterResp, error) {
	var (
		openRouterResponse external_models.OpenRouterResp
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
	err = r.getNewSendRequestObject(request, headers, "").SendRequest(&openRouterResponse)
	if err != nil {
		logger.Error("open router get chat completions", openRouterResponse, err.Error())
		return openRouterResponse, err
	}
	fmt.Println(openRouterResponse)
	return openRouterResponse, nil
}
