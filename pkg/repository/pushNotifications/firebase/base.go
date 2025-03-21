package firebase

import (
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
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
