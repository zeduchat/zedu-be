package centrifuge

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func NewCentrifugoService(logger *utility.Logger, centrifugoURL string) *centrifuge.Client {
	client := centrifuge.NewJsonClient(
		centrifugoURL,
		centrifuge.Config{
			GetToken: func(_ centrifuge.ConnectionTokenEvent) (string, error) {
				utility.LogAndPrint(logger, "Referesh connection event")
				token := GetConnToken()
				return token, nil
			},
		},
	)

	if err := client.Connect(); err != nil {
		utility.LogAndPrint(logger, "an error occured while connecting to centrifugo server: ", err)
		return nil
	}

	utility.LogAndPrint(logger, "connected to centrifugo server")

	Client.Client = client

	return client
}

func (s *CentClient) BroadcastChannel(logger *utility.Logger, channelID string, broadcastPayload interface{}) error {

	payload, err := json.Marshal(broadcastPayload)
	if err != nil {
		return err
	}

	pubRes, err := s.Client.Publish(context.Background(), channelID, payload)

	if err != nil {
		utility.LogAndPrint(logger, fmt.Sprintf("Failed to publish to channel %s: %v", channelID, err))
		return err
	}

	utility.LogAndPrint(logger, fmt.Sprintf("published  %v to %s", pubRes, channelID))

	return nil
}

func GetConnToken() string {

	userClaims := jwt.MapClaims{}
	userClaims["exp"] = time.Now().Unix() + int64(30)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	connToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return connToken
	}
	return connToken
}
