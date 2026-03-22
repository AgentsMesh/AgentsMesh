package eventbus

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEntityEvent(t *testing.T) {
	t.Run("creates entity event with valid data", func(t *testing.T) {
		data := &PodStatusChangedData{
			PodKey:         "pod-123",
			Status:         "running",
			PreviousStatus: "initializing",
		}

		before := time.Now().UnixMilli()
		event, err := NewEntityEvent(EventPodStatusChanged, 1, "pod", "pod-123", data)
		after := time.Now().UnixMilli()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event == nil {
			t.Fatal("expected non-nil event")
		}

		if event.Type != EventPodStatusChanged {
			t.Errorf("expected type %s, got %s", EventPodStatusChanged, event.Type)
		}
		if event.Category != CategoryEntity {
			t.Errorf("expected category %s, got %s", CategoryEntity, event.Category)
		}
		if event.OrganizationID != 1 {
			t.Errorf("expected org ID 1, got %d", event.OrganizationID)
		}
		if event.EntityType != "pod" {
			t.Errorf("expected entity type 'pod', got '%s'", event.EntityType)
		}
		if event.EntityID != "pod-123" {
			t.Errorf("expected entity ID 'pod-123', got '%s'", event.EntityID)
		}
		if event.Timestamp < before || event.Timestamp > after {
			t.Errorf("timestamp %d not in range [%d, %d]", event.Timestamp, before, after)
		}

		// Verify data is correctly marshaled
		var decoded PodStatusChangedData
		if err := json.Unmarshal(event.Data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal data: %v", err)
		}
		if decoded.PodKey != "pod-123" {
			t.Errorf("expected PodKey 'pod-123', got '%s'", decoded.PodKey)
		}
		if decoded.Status != "running" {
			t.Errorf("expected Status 'running', got '%s'", decoded.Status)
		}
	})

	t.Run("creates event with ticket data", func(t *testing.T) {
		data := &TicketStatusChangedData{
			Slug:     "AM-001",
			Status:         "in_progress",
			PreviousStatus: "backlog",
		}

		event, err := NewEntityEvent(EventTicketStatusChanged, 42, "ticket", "AM-001", data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded TicketStatusChangedData
		if err := json.Unmarshal(event.Data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal data: %v", err)
		}
		if decoded.Slug != "AM-001" {
			t.Errorf("expected Slug 'AM-001', got '%s'", decoded.Slug)
		}
	})

	t.Run("creates event with runner data", func(t *testing.T) {
		data := &RunnerStatusData{
			RunnerID:      100,
			NodeID:        "node-abc",
			Status:        "online",
			CurrentPods:   5,
			LastHeartbeat: "2024-01-01T00:00:00Z",
		}

		event, err := NewEntityEvent(EventRunnerOnline, 1, "runner", "runner-100", data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded RunnerStatusData
		if err := json.Unmarshal(event.Data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal data: %v", err)
		}
		if decoded.RunnerID != 100 {
			t.Errorf("expected RunnerID 100, got %d", decoded.RunnerID)
		}
		if decoded.CurrentPods != 5 {
			t.Errorf("expected CurrentPods 5, got %d", decoded.CurrentPods)
		}
	})

	t.Run("creates event with map data", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		event, err := NewEntityEvent(EventPodCreated, 1, "pod", "pod-456", data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal(event.Data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal data: %v", err)
		}
		if decoded["key1"] != "value1" {
			t.Errorf("expected key1='value1', got '%v'", decoded["key1"])
		}
	})

	t.Run("creates event with nil data", func(t *testing.T) {
		event, err := NewEntityEvent(EventPodTerminated, 1, "pod", "pod-789", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// nil marshals to "null"
		if string(event.Data) != "null" {
			t.Errorf("expected 'null', got '%s'", string(event.Data))
		}
	})

	t.Run("handles unmarshalable data", func(t *testing.T) {
		// Channel cannot be marshaled to JSON
		ch := make(chan int)

		_, err := NewEntityEvent(EventPodCreated, 1, "pod", "pod-err", ch)

		if err == nil {
			t.Error("expected error for unmarshalable data")
		}
	})
}

func TestNewNotificationEvent(t *testing.T) {
	t.Run("creates notification event with single target user", func(t *testing.T) {
		userID := int64(100)
		data := &PodNotificationData{
			PodKey: "pod-123",
			Title:  "Task Complete",
			Body:   "Your task has finished",
		}

		before := time.Now().UnixMilli()
		event, err := NewNotificationEvent(EventPodNotification, 1, &userID, nil, "pod", "pod-123", data)
		after := time.Now().UnixMilli()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if event == nil {
			t.Fatal("expected non-nil event")
		}

		if event.Type != EventPodNotification {
			t.Errorf("expected type %s, got %s", EventPodNotification, event.Type)
		}
		if event.Category != CategoryNotification {
			t.Errorf("expected category %s, got %s", CategoryNotification, event.Category)
		}
		if event.TargetUserID == nil || *event.TargetUserID != 100 {
			t.Errorf("expected target user ID 100")
		}
		if event.Timestamp < before || event.Timestamp > after {
			t.Errorf("timestamp %d not in range [%d, %d]", event.Timestamp, before, after)
		}

		var decoded PodNotificationData
		if err := json.Unmarshal(event.Data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal data: %v", err)
		}
		if decoded.Title != "Task Complete" {
			t.Errorf("expected Title 'Task Complete', got '%s'", decoded.Title)
		}
	})

	t.Run("creates notification event with multiple target users", func(t *testing.T) {
		userIDs := []int64{100, 200, 300}
		data := map[string]string{"message": "You were mentioned"}

		event, err := NewNotificationEvent(EventMentionNotification, 1, nil, userIDs, "channel", "channel-1", data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.TargetUserID != nil {
			t.Error("expected nil target user ID for multi-user notification")
		}
		if len(event.TargetUserIDs) != 3 {
			t.Errorf("expected 3 target user IDs, got %d", len(event.TargetUserIDs))
		}
		for i, id := range event.TargetUserIDs {
			if id != userIDs[i] {
				t.Errorf("target user ID %d: expected %d, got %d", i, userIDs[i], id)
			}
		}
	})

	t.Run("creates notification event with task completed data", func(t *testing.T) {
		userID := int64(50)
		ticketID := int64(999)
		data := &TaskCompletedData{
			PodKey:      "pod-abc",
			AgentStatus: "completed",
			TicketID:    &ticketID,
		}

		event, err := NewNotificationEvent(EventTaskCompleted, 1, &userID, nil, "pod", "pod-abc", data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var decoded TaskCompletedData
		if err := json.Unmarshal(event.Data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal data: %v", err)
		}
		if decoded.AgentStatus != "completed" {
			t.Errorf("expected AgentStatus 'completed', got '%s'", decoded.AgentStatus)
		}
		if decoded.TicketID == nil || *decoded.TicketID != 999 {
			t.Errorf("expected TicketID 999")
		}
	})

	t.Run("creates notification event with nil target users", func(t *testing.T) {
		data := map[string]string{"info": "broadcast"}

		event, err := NewNotificationEvent(EventTaskCompleted, 1, nil, nil, "", "", data)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if event.TargetUserID != nil {
			t.Error("expected nil target user ID")
		}
		if len(event.TargetUserIDs) != 0 {
			t.Errorf("expected empty target user IDs, got %d", len(event.TargetUserIDs))
		}
	})

	t.Run("handles unmarshalable data", func(t *testing.T) {
		ch := make(chan int)
		userID := int64(1)

		_, err := NewNotificationEvent(EventPodNotification, 1, &userID, nil, "pod", "pod-err", ch)

		if err == nil {
			t.Error("expected error for unmarshalable data")
		}
	})
}

func TestEventType_Constants(t *testing.T) {
	// Verify event type constants are unique
	eventTypes := []EventType{
		EventPodCreated,
		EventPodStatusChanged,
		EventPodAgentChanged,
		EventPodTerminated,
		EventTicketCreated,
		EventTicketUpdated,
		EventTicketStatusChanged,
		EventTicketMoved,
		EventTicketDeleted,
		EventRunnerOnline,
		EventRunnerOffline,
		EventRunnerUpdated,
		EventPodNotification,
		EventTaskCompleted,
		EventMentionNotification,
		EventSystemMaintenance,
	}

	seen := make(map[EventType]bool)
	for _, et := range eventTypes {
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true
	}
}

func TestEventCategory_Constants(t *testing.T) {
	// Verify category constants are unique
	categories := []EventCategory{
		CategoryEntity,
		CategoryNotification,
		CategorySystem,
	}

	seen := make(map[EventCategory]bool)
	for _, c := range categories {
		if seen[c] {
			t.Errorf("duplicate category: %s", c)
		}
		seen[c] = true
	}
}
