package subscription

import (
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
)

func createSubscription(customerID, priceID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customerID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(priceID),
			},
		},
	}
	Usersub, err := subscription.New(params)
	if err != nil {
		return nil, err
	}
	return Usersub, nil
}

func listSubscriptions(customerID string) ([]*stripe.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
	}
	i := subscription.List(params)

	var subscriptions []*stripe.Subscription
	for i.Next() {
		subscriptions = append(subscriptions, i.Subscription())
	}

	if err := i.Err(); err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func modifySubscription(subscriptionID, newPriceID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(subscriptionID),
				Price: stripe.String(newPriceID),
			},
		},
	}
	sub, err := subscription.Update(subscriptionID, params)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

func deleteSubscription(subscriptionID string) (*stripe.Subscription, error) {
	sub, err := subscription.Cancel(subscriptionID, nil)
	if err != nil {
		return nil, err
	}
	return sub, nil
}
