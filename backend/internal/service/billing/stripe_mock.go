package billing

import (
	"fmt"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v76"
)

type MockStripeClient struct {
	mu sync.RWMutex

	customers     map[string]*stripe.Customer
	subscriptions map[string]*stripe.Subscription

	CreateCustomerErr       error
	CancelSubscriptionErr   error

	CreateCustomerCalls     []CreateCustomerCall
	CancelSubscriptionCalls []CancelSubscriptionCall

	idCounter int64
}

type CreateCustomerCall struct {
	Params *stripe.CustomerParams
	Result *stripe.Customer
	Error  error
}

type CancelSubscriptionCall struct {
	ID     string
	Params *stripe.SubscriptionCancelParams
	Result *stripe.Subscription
	Error  error
}

func NewMockStripeClient() *MockStripeClient {
	return &MockStripeClient{
		customers:     make(map[string]*stripe.Customer),
		subscriptions: make(map[string]*stripe.Subscription),
	}
}

func (m *MockStripeClient) generateID(prefix string) string {
	m.idCounter++
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), m.idCounter)
}

func (m *MockStripeClient) CreateCustomer(params *stripe.CustomerParams) (*stripe.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CreateCustomerErr != nil {
		call := CreateCustomerCall{Params: params, Error: m.CreateCustomerErr}
		m.CreateCustomerCalls = append(m.CreateCustomerCalls, call)
		return nil, m.CreateCustomerErr
	}

	customerID := m.generateID("cus")
	cust := &stripe.Customer{
		ID:      customerID,
		Email:   stripe.StringValue(params.Email),
		Name:    stripe.StringValue(params.Name),
		Created: time.Now().Unix(),
		Livemode: false,
	}

	if params.Metadata != nil {
		cust.Metadata = params.Metadata
	}

	m.customers[customerID] = cust

	call := CreateCustomerCall{Params: params, Result: cust}
	m.CreateCustomerCalls = append(m.CreateCustomerCalls, call)

	return cust, nil
}

func (m *MockStripeClient) CancelSubscription(id string, params *stripe.SubscriptionCancelParams) (*stripe.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CancelSubscriptionErr != nil {
		call := CancelSubscriptionCall{ID: id, Params: params, Error: m.CancelSubscriptionErr}
		m.CancelSubscriptionCalls = append(m.CancelSubscriptionCalls, call)
		return nil, m.CancelSubscriptionErr
	}

	sub, ok := m.subscriptions[id]
	if !ok {
		sub = &stripe.Subscription{
			ID:       id,
			Status:   stripe.SubscriptionStatusActive,
			Created:  time.Now().Unix(),
			Livemode: false,
		}
		m.subscriptions[id] = sub
	}

	sub.Status = stripe.SubscriptionStatusCanceled
	sub.CanceledAt = time.Now().Unix()
	canceledAtTime := time.Now()
	sub.EndedAt = canceledAtTime.Unix()

	call := CancelSubscriptionCall{ID: id, Params: params, Result: sub}
	m.CancelSubscriptionCalls = append(m.CancelSubscriptionCalls, call)

	return sub, nil
}

func (m *MockStripeClient) AddSubscription(sub *stripe.Subscription) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[sub.ID] = sub
}

func (m *MockStripeClient) GetCustomer(id string) (*stripe.Customer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cust, ok := m.customers[id]
	return cust, ok
}

func (m *MockStripeClient) GetSubscription(id string) (*stripe.Subscription, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sub, ok := m.subscriptions[id]
	return sub, ok
}

func (m *MockStripeClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customers = make(map[string]*stripe.Customer)
	m.subscriptions = make(map[string]*stripe.Subscription)
	m.CreateCustomerErr = nil
	m.CancelSubscriptionErr = nil
	m.CreateCustomerCalls = nil
	m.CancelSubscriptionCalls = nil
	m.idCounter = 0
}
