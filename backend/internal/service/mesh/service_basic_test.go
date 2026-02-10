package mesh

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Error("expected service.db to be the provided db")
	}
}

func TestPodToNode(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

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
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

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
	db := setupTestDB(t)
	service := NewService(db, nil, nil, nil)

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
