package eventbus

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventDataStructures(t *testing.T) {
	t.Run("PodStatusChangedData serialization", func(t *testing.T) {
		data := &PodStatusChangedData{
			PodKey:         "pod-123",
			Status:         "running",
			PreviousStatus: "pending",
			AgentStatus:    "executing",
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
			PodKey:       "pod-err-1",
			Status:       "error",
			ErrorCode:    "GIT_AUTH_FAILED",
			ErrorMessage: "authentication failed for repository",
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
		data := &PodStatusChangedData{
			PodKey: "pod-ok-1",
			Status: "running",
		}

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
			PodKey:      "pod-new",
			Status:      "initializing",
			AgentStatus: "idle",
			RunnerID:    10,
			TicketID:    &ticketID,
			CreatedByID: 5,
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
			PodKey:      "pod-no-ticket",
			Status:      "running",
			RunnerID:    1,
			TicketID:    nil,
			CreatedByID: 1,
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
			RunnerID:      99,
			NodeID:        "node-xyz",
			Status:        "offline",
			CurrentPods:   0,
			LastHeartbeat: "2024-12-01T12:00:00Z",
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
			Identifier:     "PRJ-123",
			Status:         "done",
			PreviousStatus: "in_progress",
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded TicketStatusChangedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.Identifier != "PRJ-123" {
			t.Errorf("Identifier mismatch")
		}
		if decoded.Status != "done" {
			t.Errorf("Status mismatch")
		}
		if decoded.PreviousStatus != "in_progress" {
			t.Errorf("PreviousStatus mismatch")
		}
	})

	t.Run("TerminalNotificationData serialization", func(t *testing.T) {
		data := &TerminalNotificationData{
			PodKey: "pod-term",
			Title:  "Alert",
			Body:   "Something happened",
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded TerminalNotificationData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.PodKey != "pod-term" {
			t.Errorf("PodKey mismatch")
		}
		if decoded.Title != "Alert" {
			t.Errorf("Title mismatch")
		}
		if decoded.Body != "Something happened" {
			t.Errorf("Body mismatch")
		}
	})

	t.Run("TaskCompletedData serialization", func(t *testing.T) {
		ticketID := int64(500)
		data := &TaskCompletedData{
			PodKey:      "pod-task",
			AgentStatus: "failed",
			TicketID:    &ticketID,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded TaskCompletedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.PodKey != "pod-task" {
			t.Errorf("PodKey mismatch")
		}
		if decoded.AgentStatus != "failed" {
			t.Errorf("AgentStatus mismatch")
		}
		if decoded.TicketID == nil || *decoded.TicketID != 500 {
			t.Error("TicketID mismatch")
		}
	})

	t.Run("TaskCompletedData with nil TicketID", func(t *testing.T) {
		data := &TaskCompletedData{
			PodKey:      "pod-no-ticket",
			AgentStatus: "completed",
			TicketID:    nil,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded TaskCompletedData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.TicketID != nil {
			t.Error("expected nil TicketID")
		}
	})
}

func TestEvent_Serialization(t *testing.T) {
	t.Run("full event serialization", func(t *testing.T) {
		userID := int64(42)
		userIDs := []int64{1, 2, 3}
		data, _ := json.Marshal(map[string]string{"key": "value"})

		event := &Event{
			Type:             EventPodCreated,
			Category:         CategoryEntity,
			OrganizationID:   100,
			TargetUserID:     &userID,
			TargetUserIDs:    userIDs,
			EntityType:       "pod",
			EntityID:         "pod-123",
			Data:             data,
			Timestamp:        1234567890,
			SourceInstanceID: "server-1",
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
		if decoded.TargetUserID == nil || *decoded.TargetUserID != 42 {
			t.Error("TargetUserID mismatch")
		}
		if len(decoded.TargetUserIDs) != 3 {
			t.Errorf("TargetUserIDs length mismatch")
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
			Type:           EventTicketUpdated,
			Category:       CategoryEntity,
			OrganizationID: 1,
			Timestamp:      1000,
		}

		bytes, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		// Verify omitempty works
		jsonStr := string(bytes)
		if containsSubstr(jsonStr, "target_user_id") {
			t.Error("expected target_user_id to be omitted")
		}
		if containsSubstr(jsonStr, "target_user_ids") {
			t.Error("expected target_user_ids to be omitted")
		}
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

// ===========================================
// MREventData Tests
// ===========================================

func TestMREventData_Serialization(t *testing.T) {
	t.Run("full MREventData serialization", func(t *testing.T) {
		ticketID := int64(100)
		podID := int64(200)
		data := &MREventData{
			MRID:           1,
			MRIID:          42,
			MRURL:          "https://gitlab.com/org/repo/-/merge_requests/42",
			SourceBranch:   "feature/AM-100-new-feature",
			TargetBranch:   "main",
			Title:          "Add new feature",
			State:          "opened",
			Action:         "opened",
			TicketID:       &ticketID,
			PodID:          &podID,
			RepositoryID:   500,
			PipelineStatus: "pending",
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded MREventData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.MRID != 1 {
			t.Errorf("MRID mismatch: expected 1, got %d", decoded.MRID)
		}
		if decoded.MRIID != 42 {
			t.Errorf("MRIID mismatch: expected 42, got %d", decoded.MRIID)
		}
		if decoded.MRURL != "https://gitlab.com/org/repo/-/merge_requests/42" {
			t.Errorf("MRURL mismatch: %s", decoded.MRURL)
		}
		if decoded.SourceBranch != "feature/AM-100-new-feature" {
			t.Errorf("SourceBranch mismatch: %s", decoded.SourceBranch)
		}
		if decoded.TargetBranch != "main" {
			t.Errorf("TargetBranch mismatch: %s", decoded.TargetBranch)
		}
		if decoded.Title != "Add new feature" {
			t.Errorf("Title mismatch: %s", decoded.Title)
		}
		if decoded.State != "opened" {
			t.Errorf("State mismatch: %s", decoded.State)
		}
		if decoded.Action != "opened" {
			t.Errorf("Action mismatch: %s", decoded.Action)
		}
		if decoded.TicketID == nil || *decoded.TicketID != 100 {
			t.Error("TicketID mismatch")
		}
		if decoded.PodID == nil || *decoded.PodID != 200 {
			t.Error("PodID mismatch")
		}
		if decoded.RepositoryID != 500 {
			t.Errorf("RepositoryID mismatch: expected 500, got %d", decoded.RepositoryID)
		}
		if decoded.PipelineStatus != "pending" {
			t.Errorf("PipelineStatus mismatch: %s", decoded.PipelineStatus)
		}
	})

	t.Run("MREventData with nil optional fields", func(t *testing.T) {
		data := &MREventData{
			MRID:         1,
			MRIID:        10,
			MRURL:        "https://github.com/org/repo/pull/10",
			SourceBranch: "fix-bug",
			State:        "merged",
			RepositoryID: 100,
			// TargetBranch, Title, Action, TicketID, PodID, PipelineStatus are omitted
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded MREventData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.TicketID != nil {
			t.Error("expected nil TicketID")
		}
		if decoded.PodID != nil {
			t.Error("expected nil PodID")
		}
		if decoded.TargetBranch != "" {
			t.Errorf("expected empty TargetBranch, got %s", decoded.TargetBranch)
		}
		if decoded.Title != "" {
			t.Errorf("expected empty Title, got %s", decoded.Title)
		}
		if decoded.Action != "" {
			t.Errorf("expected empty Action, got %s", decoded.Action)
		}
		if decoded.PipelineStatus != "" {
			t.Errorf("expected empty PipelineStatus, got %s", decoded.PipelineStatus)
		}
	})

	t.Run("MREventData JSON omitempty behavior", func(t *testing.T) {
		data := &MREventData{
			MRID:         1,
			MRIID:        5,
			MRURL:        "https://gitlab.com/mr/5",
			SourceBranch: "dev",
			State:        "closed",
			RepositoryID: 10,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		jsonStr := string(bytes)
		// Fields with omitempty should not appear when empty/nil
		if containsSubstr(jsonStr, "target_branch") {
			t.Error("expected target_branch to be omitted when empty")
		}
		if containsSubstr(jsonStr, "title") {
			t.Error("expected title to be omitted when empty")
		}
		if containsSubstr(jsonStr, "action") {
			t.Error("expected action to be omitted when empty")
		}
		if containsSubstr(jsonStr, "ticket_id") {
			t.Error("expected ticket_id to be omitted when nil")
		}
		if containsSubstr(jsonStr, "pod_id") {
			t.Error("expected pod_id to be omitted when nil")
		}
		if containsSubstr(jsonStr, "pipeline_status") {
			t.Error("expected pipeline_status to be omitted when empty")
		}
	})

	t.Run("MREventData all states", func(t *testing.T) {
		states := []string{"opened", "merged", "closed"}
		actions := []string{"opened", "updated", "merged", "closed"}

		for _, state := range states {
			for _, action := range actions {
				data := &MREventData{
					MRID:         1,
					MRIID:        1,
					MRURL:        "https://example.com/mr/1",
					SourceBranch: "test",
					State:        state,
					Action:       action,
					RepositoryID: 1,
				}

				bytes, err := json.Marshal(data)
				if err != nil {
					t.Fatalf("failed to marshal state=%s action=%s: %v", state, action, err)
				}

				var decoded MREventData
				if err := json.Unmarshal(bytes, &decoded); err != nil {
					t.Fatalf("failed to unmarshal state=%s action=%s: %v", state, action, err)
				}

				if decoded.State != state {
					t.Errorf("State mismatch: expected %s, got %s", state, decoded.State)
				}
				if decoded.Action != action {
					t.Errorf("Action mismatch: expected %s, got %s", action, decoded.Action)
				}
			}
		}
	})
}

// ===========================================
// PipelineEventData Tests
// ===========================================

func TestPipelineEventData_Serialization(t *testing.T) {
	t.Run("full PipelineEventData serialization", func(t *testing.T) {
		ticketID := int64(50)
		podID := int64(60)
		data := &PipelineEventData{
			MRID:           10,
			PipelineID:     12345,
			PipelineStatus: "success",
			PipelineURL:    "https://gitlab.com/org/repo/-/pipelines/12345",
			SourceBranch:   "feature/new-feature",
			TicketID:       &ticketID,
			PodID:          &podID,
			RepositoryID:   300,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded PipelineEventData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.MRID != 10 {
			t.Errorf("MRID mismatch: expected 10, got %d", decoded.MRID)
		}
		if decoded.PipelineID != 12345 {
			t.Errorf("PipelineID mismatch: expected 12345, got %d", decoded.PipelineID)
		}
		if decoded.PipelineStatus != "success" {
			t.Errorf("PipelineStatus mismatch: %s", decoded.PipelineStatus)
		}
		if decoded.PipelineURL != "https://gitlab.com/org/repo/-/pipelines/12345" {
			t.Errorf("PipelineURL mismatch: %s", decoded.PipelineURL)
		}
		if decoded.SourceBranch != "feature/new-feature" {
			t.Errorf("SourceBranch mismatch: %s", decoded.SourceBranch)
		}
		if decoded.TicketID == nil || *decoded.TicketID != 50 {
			t.Error("TicketID mismatch")
		}
		if decoded.PodID == nil || *decoded.PodID != 60 {
			t.Error("PodID mismatch")
		}
		if decoded.RepositoryID != 300 {
			t.Errorf("RepositoryID mismatch: expected 300, got %d", decoded.RepositoryID)
		}
	})

	t.Run("PipelineEventData with nil optional fields", func(t *testing.T) {
		data := &PipelineEventData{
			PipelineID:     999,
			PipelineStatus: "failed",
			RepositoryID:   100,
			// MRID, PipelineURL, SourceBranch, TicketID, PodID are omitted
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded PipelineEventData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.MRID != 0 {
			t.Errorf("expected zero MRID, got %d", decoded.MRID)
		}
		if decoded.PipelineURL != "" {
			t.Errorf("expected empty PipelineURL, got %s", decoded.PipelineURL)
		}
		if decoded.SourceBranch != "" {
			t.Errorf("expected empty SourceBranch, got %s", decoded.SourceBranch)
		}
		if decoded.TicketID != nil {
			t.Error("expected nil TicketID")
		}
		if decoded.PodID != nil {
			t.Error("expected nil PodID")
		}
	})

	t.Run("PipelineEventData JSON omitempty behavior", func(t *testing.T) {
		data := &PipelineEventData{
			PipelineID:     100,
			PipelineStatus: "running",
			RepositoryID:   1,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		jsonStr := string(bytes)
		// Fields with omitempty should not appear when empty/nil
		if containsSubstr(jsonStr, "mr_id") {
			t.Error("expected mr_id to be omitted when zero")
		}
		if containsSubstr(jsonStr, "pipeline_url") {
			t.Error("expected pipeline_url to be omitted when empty")
		}
		if containsSubstr(jsonStr, "source_branch") {
			t.Error("expected source_branch to be omitted when empty")
		}
		if containsSubstr(jsonStr, "ticket_id") {
			t.Error("expected ticket_id to be omitted when nil")
		}
		if containsSubstr(jsonStr, "pod_id") {
			t.Error("expected pod_id to be omitted when nil")
		}
	})

	t.Run("PipelineEventData all statuses", func(t *testing.T) {
		statuses := []string{"pending", "running", "success", "failed", "canceled", "skipped"}

		for _, status := range statuses {
			data := &PipelineEventData{
				PipelineID:     1,
				PipelineStatus: status,
				RepositoryID:   1,
			}

			bytes, err := json.Marshal(data)
			if err != nil {
				t.Fatalf("failed to marshal status=%s: %v", status, err)
			}

			var decoded PipelineEventData
			if err := json.Unmarshal(bytes, &decoded); err != nil {
				t.Fatalf("failed to unmarshal status=%s: %v", status, err)
			}

			if decoded.PipelineStatus != status {
				t.Errorf("PipelineStatus mismatch: expected %s, got %s", status, decoded.PipelineStatus)
			}
		}
	})

	t.Run("PipelineEventData without MR association", func(t *testing.T) {
		// Pipeline can exist without being associated with an MR
		data := &PipelineEventData{
			PipelineID:     5000,
			PipelineStatus: "success",
			PipelineURL:    "https://gitlab.com/org/repo/-/pipelines/5000",
			SourceBranch:   "main",
			RepositoryID:   10,
			// MRID is 0 (no MR association)
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded PipelineEventData
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.MRID != 0 {
			t.Errorf("expected zero MRID for pipeline without MR, got %d", decoded.MRID)
		}
		if decoded.PipelineID != 5000 {
			t.Errorf("PipelineID mismatch")
		}
	})
}

// ===========================================
// MR/Pipeline Event Type Constants Tests
// ===========================================

func TestMREventTypes(t *testing.T) {
	t.Run("MR event type constants", func(t *testing.T) {
		if EventMRCreated != "mr:created" {
			t.Errorf("unexpected EventMRCreated: %s", EventMRCreated)
		}
		if EventMRUpdated != "mr:updated" {
			t.Errorf("unexpected EventMRUpdated: %s", EventMRUpdated)
		}
		if EventMRMerged != "mr:merged" {
			t.Errorf("unexpected EventMRMerged: %s", EventMRMerged)
		}
		if EventMRClosed != "mr:closed" {
			t.Errorf("unexpected EventMRClosed: %s", EventMRClosed)
		}
	})

	t.Run("Pipeline event type constant", func(t *testing.T) {
		if EventPipelineUpdated != "pipeline:updated" {
			t.Errorf("unexpected EventPipelineUpdated: %s", EventPipelineUpdated)
		}
	})
}

// ===========================================
// Event with MREventData/PipelineEventData Tests
// ===========================================

func TestEvent_WithMREventData(t *testing.T) {
	t.Run("Event with MREventData payload", func(t *testing.T) {
		mrData := &MREventData{
			MRID:         1,
			MRIID:        42,
			MRURL:        "https://gitlab.com/mr/42",
			SourceBranch: "feature-branch",
			State:        "opened",
			RepositoryID: 100,
		}

		mrDataBytes, _ := json.Marshal(mrData)

		event := &Event{
			Type:           EventMRCreated,
			Category:       CategoryEntity,
			OrganizationID: 1,
			EntityType:     "merge_request",
			EntityID:       "1",
			Data:           mrDataBytes,
			Timestamp:      1234567890,
		}

		bytes, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		var decoded Event
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if decoded.Type != EventMRCreated {
			t.Errorf("Type mismatch: expected %s, got %s", EventMRCreated, decoded.Type)
		}
		if decoded.EntityType != "merge_request" {
			t.Errorf("EntityType mismatch")
		}

		// Verify embedded MREventData can be extracted
		var extractedMRData MREventData
		if err := json.Unmarshal(decoded.Data, &extractedMRData); err != nil {
			t.Fatalf("failed to unmarshal MREventData from event: %v", err)
		}

		if extractedMRData.MRIID != 42 {
			t.Errorf("MRIID mismatch in extracted data")
		}
		if extractedMRData.SourceBranch != "feature-branch" {
			t.Errorf("SourceBranch mismatch in extracted data")
		}
	})

	t.Run("Event with PipelineEventData payload", func(t *testing.T) {
		pipelineData := &PipelineEventData{
			PipelineID:     999,
			PipelineStatus: "success",
			PipelineURL:    "https://gitlab.com/pipeline/999",
			RepositoryID:   50,
		}

		pipelineDataBytes, _ := json.Marshal(pipelineData)

		event := &Event{
			Type:           EventPipelineUpdated,
			Category:       CategoryEntity,
			OrganizationID: 2,
			EntityType:     "pipeline",
			EntityID:       "999",
			Data:           pipelineDataBytes,
			Timestamp:      1234567890,
		}

		bytes, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		var decoded Event
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}

		if decoded.Type != EventPipelineUpdated {
			t.Errorf("Type mismatch: expected %s, got %s", EventPipelineUpdated, decoded.Type)
		}

		// Verify embedded PipelineEventData can be extracted
		var extractedPipelineData PipelineEventData
		if err := json.Unmarshal(decoded.Data, &extractedPipelineData); err != nil {
			t.Fatalf("failed to unmarshal PipelineEventData from event: %v", err)
		}

		if extractedPipelineData.PipelineID != 999 {
			t.Errorf("PipelineID mismatch in extracted data")
		}
		if extractedPipelineData.PipelineStatus != "success" {
			t.Errorf("PipelineStatus mismatch in extracted data")
		}
	})
}
