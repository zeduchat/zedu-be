package request

import (
	"fmt"
	"net/http"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/mocks"
	"github.com/hngprojects/telex_be/external/thirdparty/integrations"
	"github.com/hngprojects/telex_be/external/thirdparty/ipstack"
	"github.com/hngprojects/telex_be/external/thirdparty/openrouter"
	"github.com/hngprojects/telex_be/external/thirdparty/slack"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

type ExternalRequest struct {
	Logger *utility.Logger
	Test   bool
}

var (
	JsonDecodeMethod    string = "json"
	PhpSerializerMethod string = "phpserializer"
	IpinfoResolveIp     string = "ipinfo_resolve_ip"
	SlackOAuthExchange  string = "slack_oauth_exchange"
	SlackGetChannels    string = "slack_get_channels"
	SlackGetManifest    string = "slack_get_manifest"
	SlackGetAccessToken string = "slack_get_access_token"
	AgentJsonContent    string = "fetch_agent_json_content"
	GetChatCompletions  string = "get_open_router_chat_completions"
	GetAllModels        string = "get_all_models"
	SendAgentAPIKey     string = "send_agent_api_key"
)

func (er ExternalRequest) SendExternalRequest(name string, data any) (any, error) {
	var (
		config = config.GetConfig()
	)
	if !er.Test {
		switch name {
		case IpinfoResolveIp:
			obj := ipstack.RequestObj{
				Name:         name,
				Path:         fmt.Sprintf("%v", config.IPStack.BaseUrl),
				Method:       http.MethodGet,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
			}
			return obj.IpinfoResolveIp()
		case SlackOAuthExchange:
			obj := slack.RequestObj{
				Name:         name,
				Path:         fmt.Sprintf("%v", config.Slack.BaseUrl),
				Method:       http.MethodPost,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
			}
			return obj.ExchangeSlackOAuthToken()
		case SlackGetChannels:
			obj := slack.RequestObj{
				Name:         name,
				Path:         fmt.Sprintf("%v", config.Slack.BaseUrl),
				Method:       http.MethodGet,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
			}
			return obj.GetSlackChannels()
		case SlackGetManifest:
			token := data.(string)

			obj := slack.RequestObj{
				Name:         name,
				Path:         fmt.Sprintf("%v", config.Slack.ManifestUrl),
				Method:       http.MethodGet,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
			}
			return obj.GetManifest(token)

		case SlackGetAccessToken:
			refresh_token := data.(string)

			obj := slack.RequestObj{
				Name:         name,
				Path:         fmt.Sprintf("%v", config.Slack.BaseUrl),
				Method:       http.MethodPost,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
			}
			return obj.GetSlackToken(refresh_token)

		case AgentJsonContent:

			data_content := data.(map[string]string)

			obj := integrations.RequestObj{
				Name:         name,
				Path:         data_content["url"],
				Method:       http.MethodGet,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
				Timeout:      true,
			}
			return obj.RetriveJsonData()
		case SendAgentAPIKey:
			data_content := data.(map[string]any)
			obj := integrations.RequestObj{
				Name:         name,
				Path:         data_content["url"].(string),
				Method:       http.MethodPost,
				SuccessCode:  200,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data_content["payload"].(map[string]string),
				Logger:       er.Logger,
			}
			return obj.SendAgentApiKey()
		case GetChatCompletions:

			obj := openrouter.RequestObj{
				Name:         name,
				Path:         config.OpenRouter.BaseUrl,
				Method:       http.MethodPost,
				SuccessCode:  http.StatusOK,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data.(external_models.OpenRouterReq),
				Logger:       er.Logger,
				Timeout:      true,
			}
			return obj.GetChatCompletions()
		case GetAllModels:
			obj := openrouter.RequestObj{
				Name:         name,
				Path:         config.OpenRouter.BaseUrl,
				Method:       http.MethodGet,
				SuccessCode:  http.StatusOK,
				DecodeMethod: JsonDecodeMethod,
				RequestData:  data,
				Logger:       er.Logger,
				Timeout:      true,
			}
			return obj.GetAllModels()
		default:
			return nil, fmt.Errorf("request not found")
		}
	} else {
		mer := mocks.ExternalRequest{Logger: er.Logger, Test: true}
		return mer.SendExternalRequest(name, data)
	}
}
