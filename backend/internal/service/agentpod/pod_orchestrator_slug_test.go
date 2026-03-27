package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// ==================== Step 3.5: TicketSlug → TicketID Resolution ====================

func TestSlugResolution_ResolvesTicketIDFromSlug(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{
		ticket: &ticket.Ticket{
			ID:   42,
			Slug: "AM-42",
		},
	}
	coord := &mockPodCoordinator{}
	orch, podSvc, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentSlug := "claude-code"
	ticketSlug := "AM-42"
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		TicketSlug:     &ticketSlug,
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)

	// Verify the DB record has the resolved TicketID
	require.NotNil(t, result.Pod.TicketID, "DB record should have resolved TicketID")
	assert.Equal(t, int64(42), *result.Pod.TicketID)

	// Double-check by reading from DB
	pod, err := podSvc.GetPodByKey(context.Background(), result.Pod.PodKey)
	require.NoError(t, err)
	require.NotNil(t, pod.TicketID)
	assert.Equal(t, int64(42), *pod.TicketID)
}

func TestSlugResolution_FailureDoesNotBlockPodCreation(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{err: errors.New("ticket not found")}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentSlug := "claude-code"
	ticketSlug := "NONEXIST-999"
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		TicketSlug:     &ticketSlug,
	})

	require.NoError(t, err, "Slug resolution failure should not block pod creation")
	assert.True(t, coord.createPodCalled)
	assert.Nil(t, result.Pod.TicketID, "TicketID should remain nil when slug resolution fails")
}

func TestSlugResolution_ExplicitTicketIDTakesPriority(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{
		ticket: &ticket.Ticket{
			ID:   99, // This ID would be returned by GetTicketBySlug
			Slug: "AM-99",
		},
	}
	coord := &mockPodCoordinator{}
	orch, _, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentSlug := "claude-code"
	ticketID := int64(7) // Explicitly provided TicketID
	ticketSlug := "AM-99"
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		TicketID:       &ticketID,
		TicketSlug:     &ticketSlug,
	})

	require.NoError(t, err)
	// The explicit TicketID (7) should be used, not the resolved one (99)
	require.NotNil(t, result.Pod.TicketID)
	assert.Equal(t, int64(7), *result.Pod.TicketID, "Explicit TicketID should take priority over slug resolution")
}
