package config

type Stripe struct {
	STRIPE_KEY            string
	STRIPE_WEBHOOK_SECRET string
	STRIPE_BASIC_ID       string
	STRIPE_PREMIUM_ID     string
	STRIPE_ADVANCED_ID    string
}
