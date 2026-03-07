package relay

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// === Tests for DNS CRUD operations with mock provider ===

func TestDNSService_CreateRecord(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	err := svc.CreateRecord(ctx, "us-east-1", "192.168.1.1")

	assert.NoError(t, err)
	assert.Equal(t, 1, mockProvider.createCalled)
	assert.Equal(t, "192.168.1.1", mockProvider.records["us-east-1.relay.agentsmesh.cn"])
}

func TestDNSService_CreateRecord_Disabled(t *testing.T) {
	svc := newTestDNSService(nil, false)

	ctx := context.Background()
	err := svc.CreateRecord(ctx, "us-east-1", "192.168.1.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dns service is not enabled")
}

func TestDNSService_CreateRecord_ProviderError(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.createErr = errors.New("provider error")
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	err := svc.CreateRecord(ctx, "us-east-1", "192.168.1.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create DNS record")
}

func TestDNSService_DeleteRecord(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.records["us-east-1.relay.agentsmesh.cn"] = "192.168.1.1"
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	err := svc.DeleteRecord(ctx, "us-east-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, mockProvider.deleteCalled)
	assert.Empty(t, mockProvider.records["us-east-1.relay.agentsmesh.cn"])
}

func TestDNSService_DeleteRecord_Disabled(t *testing.T) {
	svc := newTestDNSService(nil, false)

	ctx := context.Background()
	err := svc.DeleteRecord(ctx, "us-east-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dns service is not enabled")
}

func TestDNSService_DeleteRecord_ProviderError(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.deleteErr = errors.New("provider error")
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	err := svc.DeleteRecord(ctx, "us-east-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete DNS record")
}

func TestDNSService_UpdateRecord(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.records["us-east-1.relay.agentsmesh.cn"] = "192.168.1.1"
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	err := svc.UpdateRecord(ctx, "us-east-1", "192.168.1.2")

	assert.NoError(t, err)
	assert.Equal(t, 1, mockProvider.updateCalled)
	assert.Equal(t, "192.168.1.2", mockProvider.records["us-east-1.relay.agentsmesh.cn"])
}

func TestDNSService_UpdateRecord_Disabled(t *testing.T) {
	svc := newTestDNSService(nil, false)

	ctx := context.Background()
	err := svc.UpdateRecord(ctx, "us-east-1", "192.168.1.2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dns service is not enabled")
}

func TestDNSService_UpdateRecord_ProviderError(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.updateErr = errors.New("provider error")
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	err := svc.UpdateRecord(ctx, "us-east-1", "192.168.1.2")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update DNS record")
}

func TestDNSService_GetRecord(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.records["us-east-1.relay.agentsmesh.cn"] = "192.168.1.1"
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	ip, err := svc.GetRecord(ctx, "us-east-1")

	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.1", ip)
	assert.Equal(t, 1, mockProvider.getCalled)
}

func TestDNSService_GetRecord_NotFound(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	ip, err := svc.GetRecord(ctx, "non-existent")

	assert.NoError(t, err)
	assert.Empty(t, ip)
}

func TestDNSService_GetRecord_Disabled(t *testing.T) {
	svc := newTestDNSService(nil, false)

	ctx := context.Background()
	ip, err := svc.GetRecord(ctx, "us-east-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dns service is not enabled")
	assert.Empty(t, ip)
}

func TestDNSService_GetRecord_ProviderError(t *testing.T) {
	mockProvider := NewMockDNSProvider()
	mockProvider.getErr = errors.New("provider error")
	svc := newTestDNSService(mockProvider, true)

	ctx := context.Background()
	ip, err := svc.GetRecord(ctx, "us-east-1")

	assert.Error(t, err)
	assert.Empty(t, ip)
}

// Test NewDNSService with disabled configuration
func TestNewDNSService_Disabled(t *testing.T) {
	// Create a disabled service directly
	svc := newTestDNSService(nil, false)

	assert.False(t, svc.IsEnabled())
}
