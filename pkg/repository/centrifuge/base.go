package centrifuge

import (
	"github.com/centrifugal/gocent"
)

type CentClient struct {
	C *gocent.Client
}

var Client *CentClient = &CentClient{}

func Connection() *CentClient {
	return Client
}
