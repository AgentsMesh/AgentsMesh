package grpc

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// MCP adapters short-circuit on invalid identifier/display payload before
// touching downstream services. Drive them with a bare *GRPCRunnerAdapter
// and nil dependencies; if validation regresses (caller-trust mode), the
// test panics on nil podOrchestrator/channelService/ticketService.

func TestMcpCreatePod_RejectsDotInAgentSlug(t *testing.T) {
	a := &GRPCRunnerAdapter{}
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 1}
	payload := []byte(`{"agent_slug":"my.bad.agent","runner_id":1}`)

	_, mcpErr := a.mcpCreatePod(context.Background(), tc, payload)
	if mcpErr == nil {
		t.Fatal("expected mcpError for slug with dot")
	}
	if mcpErr.code != 400 {
		t.Errorf("code = %d, want 400", mcpErr.code)
	}
}

func TestMcpCreatePod_RejectsEmptyAgentSlug(t *testing.T) {
	a := &GRPCRunnerAdapter{}
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 1}
	payload := []byte(`{"agent_slug":"","runner_id":1}`)

	_, mcpErr := a.mcpCreatePod(context.Background(), tc, payload)
	if mcpErr == nil || mcpErr.code != 400 {
		t.Fatalf("expected 400 for empty agent_slug, got %v", mcpErr)
	}
}

func TestMcpCreateChannel_RejectsEmptyName(t *testing.T) {
	a := &GRPCRunnerAdapter{}
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 1}
	payload := []byte(`{"name":""}`)

	_, mcpErr := a.mcpCreateChannel(context.Background(), tc, "1-standalone-abcd0001", payload)
	if mcpErr == nil || mcpErr.code != 400 {
		t.Fatalf("expected 400 for empty name, got %v", mcpErr)
	}
}

func TestMcpCreateChannel_RejectsZeroWidthOnlyName(t *testing.T) {
	// "​" (ZWSP) sanitizes to empty → displaykit returns ErrEmpty.
	a := &GRPCRunnerAdapter{}
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 1}
	payload := []byte(`{"name":"​​"}`)

	_, mcpErr := a.mcpCreateChannel(context.Background(), tc, "1-standalone-abcd0002", payload)
	if mcpErr == nil || mcpErr.code != 400 {
		t.Fatalf("expected 400 for zero-width-only name, got %v", mcpErr)
	}
}

func TestMcpCreateTicket_RejectsEmptyTitle(t *testing.T) {
	a := &GRPCRunnerAdapter{}
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 1}
	payload := []byte(`{"title":""}`)

	_, mcpErr := a.mcpCreateTicket(context.Background(), tc, payload)
	if mcpErr == nil || mcpErr.code != 400 {
		t.Fatalf("expected 400 for empty title, got %v", mcpErr)
	}
}

func TestMcpCreateTicket_RejectsRTLOverrideOnlyTitle(t *testing.T) {
	// "‮" (RTL override) sanitizes to empty.
	a := &GRPCRunnerAdapter{}
	tc := &middleware.TenantContext{OrganizationID: 1, UserID: 1}
	payload := []byte(`{"title":"‮"}`)

	_, mcpErr := a.mcpCreateTicket(context.Background(), tc, payload)
	if mcpErr == nil || mcpErr.code != 400 {
		t.Fatalf("expected 400 for RTL-only title, got %v", mcpErr)
	}
}
