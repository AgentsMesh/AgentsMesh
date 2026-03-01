package relay

import (
	"context"
	"log/slog"
)

// MockDNSProvider implements dns.Provider interface for testing
type MockDNSProvider struct {
	records      map[string]string
	createErr    error
	deleteErr    error
	updateErr    error
	getErr       error
	createCalled int
	deleteCalled int
	updateCalled int
	getCalled    int
}

func NewMockDNSProvider() *MockDNSProvider {
	return &MockDNSProvider{
		records: make(map[string]string),
	}
}

func (m *MockDNSProvider) CreateRecord(ctx context.Context, subdomain, ip string) error {
	m.createCalled++
	if m.createErr != nil {
		return m.createErr
	}
	m.records[subdomain] = ip
	return nil
}

func (m *MockDNSProvider) DeleteRecord(ctx context.Context, subdomain string) error {
	m.deleteCalled++
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.records, subdomain)
	return nil
}

func (m *MockDNSProvider) UpdateRecord(ctx context.Context, subdomain, ip string) error {
	m.updateCalled++
	if m.updateErr != nil {
		return m.updateErr
	}
	m.records[subdomain] = ip
	return nil
}

func (m *MockDNSProvider) GetRecord(ctx context.Context, subdomain string) (string, error) {
	m.getCalled++
	if m.getErr != nil {
		return "", m.getErr
	}
	return m.records[subdomain], nil
}

// newTestDNSService creates a DNSService with mock provider for testing
func newTestDNSService(provider *MockDNSProvider, enabled bool) *DNSService {
	return &DNSService{
		provider:   provider,
		baseDomain: "relay.example.com",
		enabled:    enabled,
		logger:     slog.Default(),
	}
}
