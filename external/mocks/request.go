package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/external/mocks/ipstack_mocks"
	"github.com/hngprojects/telex_be/utility"
)

type ExternalRequest struct {
	Logger     *utility.Logger
	Test       bool
	RequestObj RequestObj
}

type RequestObj struct {
	Name         string
	Path         string
	Method       string
	Headers      map[string]string
	SuccessCode  int
	RequestData  any
	DecodeMethod string
	Logger       *utility.Logger
}

var (
	JsonDecodeMethod    string = "json"
	PhpSerializerMethod string = "phpserializer"
)

func (er ExternalRequest) SendExternalRequest(name string, data any) (any, error) {
	switch name {
	case "ipinfo_resolve_ip":
		return ipstack_mocks.IpinfoResolveIp(er.Logger, data)
	default:
		return nil, fmt.Errorf("request not found")
	}
}

func (er ExternalRequest) SendStreamingExternalRequest(name string, data any, ctx ...interface{}) (<-chan external_models.StreamChunk, error) {
	switch name {
	case "get_open_router_chat_completions":
		ch := make(chan external_models.StreamChunk)

		// Define local struct to avoid import cycle
		type LocalMessage struct {
			Role string `json:"role"`
		}
		type LocalChatReq struct {
			Messages []LocalMessage `json:"messages"`
		}

		var reqData LocalChatReq
		hasToolResult := false

		// Marshal data to JSON then unmarshal to local struct
		if jsonBytes, err := json.Marshal(data); err == nil {
			if err := json.Unmarshal(jsonBytes, &reqData); err == nil {
				for _, msg := range reqData.Messages {
					if msg.Role == "tool" {
						hasToolResult = true
						break
					}
				}
			}
		}

		go func() {
			defer close(ch)

			if hasToolResult {
				// Tool execution done, return final response
				ch <- external_models.StreamChunk{
					Data: []byte(`data: {"choices":[{"delta":{"content":"HELLO WORLD"}}]}`),
				}
				ch <- external_models.StreamChunk{
					Data: []byte("data: [DONE]"),
				}
			} else {
				// First call, return tool call
				toolCall := external_models.ToolCall{
					ID:   "call_123456",
					Type: "function",
					Function: &external_models.ToolFunction{
						Name:      "capitalize_text",
						Arguments: `{"text": "hello world"}`,
					},
				}

				toolCallJSON, _ := json.Marshal(toolCall)
				chunk1Data := fmt.Sprintf(`{"choices":[{"delta":{"tool_calls":[%s]}}]}`, toolCallJSON)

				ch <- external_models.StreamChunk{
					Data: []byte("data: " + chunk1Data),
				}

				ch <- external_models.StreamChunk{
					Data: []byte("data: [DONE]"),
				}
			}

		}()

		return ch, nil

	default:
		return nil, fmt.Errorf("streaming request not found")
	}
}
