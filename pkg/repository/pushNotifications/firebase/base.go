package firebase

import (
	"firebase.google.com/go/v4/messaging"
	firebase "firebase.google.com/go/v4"
)

type FirebaseClient struct {
	App    *firebase.App
	Client *messaging.Client
}

var Client *FirebaseClient = &FirebaseClient{}

// Initialize and return a reusable Firebase client
func Connection() *FirebaseClient {

	return Client
}
