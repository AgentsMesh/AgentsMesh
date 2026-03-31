package email

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService_ConsoleProvider(t *testing.T) {
	svc := NewService(Config{
		Provider: "console",
		BaseURL:  "https://example.com",
	})

	_, ok := svc.(*ConsoleService)
	assert.True(t, ok, "provider=console should return ConsoleService")
}

func TestNewService_EmptyResendKey(t *testing.T) {
	// When ResendKey is empty, should fall back to ConsoleService
	svc := NewService(Config{
		Provider:  "resend",
		ResendKey: "",
		BaseURL:   "https://example.com",
	})

	_, ok := svc.(*ConsoleService)
	assert.True(t, ok, "empty ResendKey should fall back to ConsoleService")
}

func TestNewService_ResendProvider(t *testing.T) {
	svc := NewService(Config{
		Provider:    "resend",
		ResendKey:   "re_test_fake",
		FromAddress: "noreply@example.com",
		BaseURL:     "https://example.com",
	})

	_, ok := svc.(*ResendService)
	assert.True(t, ok, "provider=resend with key should return ResendService")
}

func TestConsoleService_SendVerificationEmail(t *testing.T) {
	svc := &ConsoleService{baseURL: "https://test.local"}

	err := svc.SendVerificationEmail(context.Background(), "user@test.com", "tok-verify-123")
	require.NoError(t, err)
}

func TestConsoleService_SendPasswordResetEmail(t *testing.T) {
	svc := &ConsoleService{baseURL: "https://test.local"}

	err := svc.SendPasswordResetEmail(context.Background(), "user@test.com", "tok-reset-456")
	require.NoError(t, err)
}

func TestConsoleService_SendOrgInvitationEmail(t *testing.T) {
	svc := &ConsoleService{baseURL: "https://test.local"}

	err := svc.SendOrgInvitationEmail(
		context.Background(),
		"invitee@test.com",
		"Acme Corp",
		"Alice",
		"tok-invite-789",
	)
	require.NoError(t, err)
}

func TestConsoleService_SendRenewalReminder(t *testing.T) {
	svc := &ConsoleService{baseURL: "https://test.local"}

	err := svc.SendRenewalReminder(
		context.Background(),
		"admin@test.com",
		"Acme Corp",
		"Pro",
		time.Now().Add(72*time.Hour),
		3,
		"acme-corp",
	)
	require.NoError(t, err)
}

func TestNewService_EmptyProvider(t *testing.T) {
	// Empty provider + empty key should default to ConsoleService
	svc := NewService(Config{
		Provider: "",
		BaseURL:  "https://example.com",
	})

	_, ok := svc.(*ConsoleService)
	assert.True(t, ok, "empty provider should default to ConsoleService")
}
