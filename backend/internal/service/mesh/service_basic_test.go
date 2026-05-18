package mesh

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	meshDomain "github.com/anthropics/agentsmesh/backend/internal/domain/mesh"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	podService "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.repo == nil {
		t.Error("expected service.repo to be set")
	}
}

func TestPodToNode(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)

	ticketID := int64(100)
	repoID := int64(200)
	model := "claude-3-sonnet"
	pod := &agentpod.Pod{
		PodKey:       "test-pod-key",
		Status:       "running",
		AgentStatus:  "executing",
		Model:        &model,
		TicketID:     &ticketID,
		RepositoryID: &repoID,
		CreatedByID:  1,
		RunnerID:     5,
	}

	node := service.podToNode(pod)

	if node.PodKey != "test-pod-key" {
		t.Errorf("PodKey = %s, want test-pod-key", node.PodKey)
	}
	if node.Status != "running" {
		t.Errorf("Status = %s, want running", node.Status)
	}
	if node.AgentStatus != "executing" {
		t.Errorf("AgentStatus = %s, want executing", node.AgentStatus)
	}
	if node.Model == nil || *node.Model != "claude-3-sonnet" {
		t.Errorf("Model mismatch")
	}
	if node.TicketID == nil || *node.TicketID != 100 {
		t.Error("TicketID mismatch")
	}
	if node.RepositoryID == nil || *node.RepositoryID != 200 {
		t.Error("RepositoryID mismatch")
	}
}

func TestPodToNode_NilValues(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)

	// Test with minimal pod (nil optional fields)
	pod := &agentpod.Pod{
		PodKey:      "minimal-pod",
		Status:      "pending",
		AgentStatus: "idle",
		CreatedByID: 1,
	}

	node := service.podToNode(pod)

	if node.PodKey != "minimal-pod" {
		t.Errorf("PodKey = %s, want minimal-pod", node.PodKey)
	}
	if node.Model != nil {
		t.Error("expected Model to be nil")
	}
	if node.TicketID != nil {
		t.Error("expected TicketID to be nil")
	}
	if node.RepositoryID != nil {
		t.Error("expected RepositoryID to be nil")
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrTicketNotFound == nil {
		t.Error("ErrTicketNotFound should not be nil")
	}
	if ErrRunnerNotFound == nil {
		t.Error("ErrRunnerNotFound should not be nil")
	}
}

func TestServiceFields(t *testing.T) {
	repo, _ := setupTestRepo(t)
	service := NewService(repo, nil, nil, nil)

	// Verify nil services are accepted
	if service.podService != nil {
		t.Error("expected podService to be nil")
	}
	if service.channelService != nil {
		t.Error("expected channelService to be nil")
	}
	if service.bindingService != nil {
		t.Error("expected bindingService to be nil")
	}
}

func TestCreatePodForTicket_DefaultsLegacyClaudeFields(t *testing.T) {
	repo, db := setupTestRepo(t)
	ps := podService.NewPodService(infra.NewPodRepository(db))
	service := NewService(repo, ps, nil, nil)

	ticketID := testkit.CreateTicket(t, db, 1, 1, "default-fields-test")

	// Caller (handler) does not provide Model/PermissionMode.
	pod, err := service.CreatePodForTicket(context.Background(), &meshDomain.CreatePodForTicketRequest{
		OrganizationID: 1,
		RunnerID:       1,
		TicketID:       ticketID,
		CreatedByID:    1,
		Prompt:         "do something",
	})
	require.NoError(t, err)
	require.NotNil(t, pod)

	// Mesh service must compensate for PodService no longer auto-defaulting.
	require.NotNil(t, pod.Model)
	assert.Equal(t, "opus", *pod.Model)
	require.NotNil(t, pod.PermissionMode)
	assert.Equal(t, agentpod.PermissionModeBypass, *pod.PermissionMode)
	assert.Equal(t, "claude-code", pod.AgentSlug)
}

func TestCreatePodForTicket_PreservesExplicitFields(t *testing.T) {
	repo, db := setupTestRepo(t)
	ps := podService.NewPodService(infra.NewPodRepository(db))
	service := NewService(repo, ps, nil, nil)

	ticketID := testkit.CreateTicket(t, db, 1, 1, "explicit-fields-test")

	pod, err := service.CreatePodForTicket(context.Background(), &meshDomain.CreatePodForTicketRequest{
		OrganizationID: 1,
		RunnerID:       1,
		TicketID:       ticketID,
		CreatedByID:    1,
		Prompt:         "task",
		Model:          "sonnet",
		PermissionMode: "plan",
	})
	require.NoError(t, err)
	require.NotNil(t, pod.Model)
	assert.Equal(t, "sonnet", *pod.Model)
	require.NotNil(t, pod.PermissionMode)
	assert.Equal(t, "plan", *pod.PermissionMode)
}
