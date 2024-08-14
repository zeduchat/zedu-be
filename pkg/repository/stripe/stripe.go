package stripe

import (
	"fmt"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
	"github.com/stripe/stripe-go/v72/client"
)

func ConnectToStripe(logger *utility.Logger, configuration config.Stripe) *client.API {
	sc := &client.API{}
	sc.Init(configuration.STRIPE_KEY, nil)

	utility.LogAndPrint(logger, fmt.Sprintln("Stripe connected successfully"))

	return sc
}
