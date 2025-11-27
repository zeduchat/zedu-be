package agora

type AgoraClient struct {
	Service *AgoraService
}

var Client *AgoraClient = &AgoraClient{}

func Connection() *AgoraClient {
	return Client
}
