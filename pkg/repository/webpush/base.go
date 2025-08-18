package webpush

type WebPushClient struct {
	C *PushClient
}

var Client *WebPushClient = &WebPushClient{}

func Connection() *WebPushClient {
	return Client
}
