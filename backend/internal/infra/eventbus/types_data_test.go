package eventbus

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventDataStructures(t *testing.T) {
	t.Run("PodStatusChangedData serialization", func(t *testing.T) {
		data := &PodStatusChangedData{
			PodKey: "pod-123", Status: "running",
			PreviousStatus: "pending", AgentStatus: "executing",
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded PodStatusChangedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.PodKey != data.PodKey {
			t.Errorf("PodKey mismatch: expected %s, got %s", data.PodKey, decoded.PodKey)
		}
		if decoded.Status != data.Status {
			t.Errorf("Status mismatch: expected %s, got %s", data.Status, decoded.Status)
		}
		if decoded.PreviousStatus != data.PreviousStatus {
			t.Errorf("PreviousStatus mismatch: expected %s, got %s", data.PreviousStatus, decoded.PreviousStatus)
		}
		if decoded.AgentStatus != data.AgentStatus {
			t.Errorf("AgentStatus mismatch: expected %s, got %s", data.AgentStatus, decoded.AgentStatus)
		}
	})

	t.Run("PodStatusChangedData with error fields serialization", func(t *testing.T) {
		data := &PodStatusChangedData{
			PodKey: "pod-err-1", Status: "error",
			ErrorCode: "GIT_AUTH_FAILED", ErrorMessage: "authentication failed for repository",
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded PodStatusChangedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.ErrorCode != "GIT_AUTH_FAILED" {
			t.Errorf("ErrorCode mismatch: expected %s, got %s", "GIT_AUTH_FAILED", decoded.ErrorCode)
		}
		if decoded.ErrorMessage != "authentication failed for repository" {
			t.Errorf("ErrorMessage mismatch: expected %s, got %s", "authentication failed for repository", decoded.ErrorMessage)
		}
	})

	t.Run("PodStatusChangedData error fields omitted when empty", func(t *testing.T) {
		data := &PodStatusChangedData{PodKey: "pod-ok-1", Status: "running"}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		jsonStr := string(bytes)
		if strings.Contains(jsonStr, "error_code") {
			t.Errorf("error_code should be omitted when empty, got: %s", jsonStr)
		}
		if strings.Contains(jsonStr, "error_message") {
			t.Errorf("error_message should be omitted when empty, got: %s", jsonStr)
		}
	})

	t.Run("PodCreatedData serialization", func(t *testing.T) {
		ticketID := int64(42)
		data := &PodCreatedData{
			PodKey: "pod-new", Status: "initializing", AgentStatus: "idle",
			RunnerID: 10, TicketID: &ticketID, CreatedByID: 5,
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded PodCreatedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.PodKey != data.PodKey {
			t.Errorf("PodKey mismatch")
		}
		if decoded.RunnerID != data.RunnerID {
			t.Errorf("RunnerID mismatch: expected %d, got %d", data.RunnerID, decoded.RunnerID)
		}
		if decoded.TicketID == nil || *decoded.TicketID != 42 {
			t.Error("TicketID mismatch")
		}
		if decoded.CreatedByID != 5 {
			t.Errorf("CreatedByID mismatch: expected 5, got %d", decoded.CreatedByID)
		}
	})

	t.Run("PodCreatedData with nil TicketID", func(t *testing.T) {
		data := &PodCreatedData{
			PodKey: "pod-no-ticket", Status: "running",
			RunnerID: 1, TicketID: nil, CreatedByID: 1,
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded PodCreatedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.TicketID != nil {
			t.Error("expected nil TicketID")
		}
	})

	t.Run("RunnerStatusData serialization", func(t *testing.T) {
		data := &RunnerStatusData{
			RunnerID: 99, NodeID: "node-xyz", Status: "offline",
			CurrentPods: 0, LastHeartbeat: "2024-12-01T12:00:00Z",
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded RunnerStatusData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.RunnerID != 99 {
			t.Errorf("RunnerID mismatch")
		}
		if decoded.NodeID != "node-xyz" {
			t.Errorf("NodeID mismatch")
		}
		if decoded.LastHeartbeat != "2024-12-01T12:00:00Z" {
			t.Errorf("LastHeartbeat mismatch")
		}
	})

	t.Run("TicketStatusChangedData serialization", func(t *testing.T) {
		data := &TicketStatusChangedData{
			Slug: "PRJ-123", Status: "done", PreviousStatus: "in_progress",
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		var decoded TicketStatusChangedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.Slug != "PRJ-123" {
			t.Errorf("Slug mismatch")
		}
		if decoded.Status != "done" {
			t.Errorf("Status mismatch")
		}
		if decoded.PreviousStatus != "in_progress" {
			t.Errorf("PreviousStatus mismatch")
		}
	})

}

func TestEvent_Serialization(t *testing.T) {
	t.Run("full event serialization", func(t *testing.T) {
		data, _ := json.Marshal(map[string]string{"key": "value"})

		event := &Event{
			Type: EventPodCreated, Category: CategoryEntity, OrganizationID: 100,
			EntityType: "pod", EntityID: "pod-123",
			Data: data, Timestamp: 1234567890, SourceInstanceID: "server-1",
		}

		bytes, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}
		var decoded Event
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}
		if decoded.Type != EventPodCreated {
			t.Errorf("Type mismatch")
		}
		if decoded.Category != CategoryEntity {
			t.Errorf("Category mismatch")
		}
		if decoded.OrganizationID != 100 {
			t.Errorf("OrganizationID mismatch")
		}
		if decoded.EntityType != "pod" {
			t.Errorf("EntityType mismatch")
		}
		if decoded.EntityID != "pod-123" {
			t.Errorf("EntityID mismatch")
		}
		if decoded.Timestamp != 1234567890 {
			t.Errorf("Timestamp mismatch")
		}
		if decoded.SourceInstanceID != "server-1" {
			t.Errorf("SourceInstanceID mismatch")
		}
	})

	t.Run("event with omitted optional fields", func(t *testing.T) {
		event := &Event{
			Type: EventTicketUpdated, Category: CategoryEntity,
			OrganizationID: 1, Timestamp: 1000,
		}
		bytes, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}
		jsonStr := string(bytes)
		if containsSubstr(jsonStr, "source_instance_id") {
			t.Error("expected source_instance_id to be omitted")
		}
	})
}

// containsSubstr checks if string contains substring
func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
