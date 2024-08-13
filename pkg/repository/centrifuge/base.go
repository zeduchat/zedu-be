package centrifuge

import "github.com/centrifugal/centrifuge-go"

type CentClient struct {
	Client *centrifuge.Client
}

var Client *CentClient = &CentClient{}

func Connection() *CentClient {
	return Client
}
