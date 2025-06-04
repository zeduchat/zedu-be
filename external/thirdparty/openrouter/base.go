package openrouter

import (
	"github.com/hngprojects/telex_be/external"
	"github.com/hngprojects/telex_be/utility"
)

type RequestObj struct {
	Name         string
	Path         string
	Method       string
	SuccessCode  int
	RequestData  interface{}
	DecodeMethod string
	Logger       *utility.Logger
	Timeout      bool
}

var (
	JsonDecodeMethod = "json"
	OpenRouterUrl    = "https://openrouter.ai/api/v1/chat/completions"
)

func (r *RequestObj) getNewSendRequestObject(data interface{}, headers map[string]string, urlprefix string) *external.SendRequestObject {
	return external.GetNewSendRequestObject(r.Logger, r.Name, r.Path, r.Method, urlprefix, r.DecodeMethod, headers, r.SuccessCode, data, r.Timeout)
}
