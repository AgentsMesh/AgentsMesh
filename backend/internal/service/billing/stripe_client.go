package billing

import (
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
)

type StripeClient interface {
	CreateCustomer(params *stripe.CustomerParams) (*stripe.Customer, error)

	CancelSubscription(id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error)
}

type DefaultStripeClient struct{}

func NewDefaultStripeClient() *DefaultStripeClient {
	return &DefaultStripeClient{}
}

func (c *DefaultStripeClient) CreateCustomer(params *stripe.CustomerParams) (*stripe.Customer, error) {
	return customer.New(params)
}

func (c *DefaultStripeClient) CancelSubscription(id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error) {
	return subscription.Cancel(id, params)
}
