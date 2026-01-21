package request

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/mocks"
	"github.com/hngprojects/telex_be/external/thirdparty/openrouter"
	"github.com/hngprojects/telex_be/internal/config"
)

func (er *ExternalRequest) SendStreamingExternalRequest(name string, data any, ctx context.Context) (<-chan external_models.StreamChunk, error) {
	var config = config.GetConfig()

	if !er.Test {
		switch name {
		case GetChatCompletions:
			obj := openrouter.RequestObj{
				Name:         name,
				Path:         fmt.Sprintf("%v", config.OpenRouter.BaseUrl),
				Method:       http.MethodPost,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
				Timeout:      true,
			}
			return obj.GetStreamChatCompletions(ctx)
		default:
			return nil, fmt.Errorf("streaming not supported for request type: %s", name)
		}
	}

	mer := mocks.ExternalRequest{Logger: er.Logger, Test: true}
	return mer.SendStreamingExternalRequest(name, data, ctx)
}
