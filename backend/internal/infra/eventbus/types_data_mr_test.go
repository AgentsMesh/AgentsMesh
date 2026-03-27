package eventbus

import (
	"encoding/json"
	"testing"
)

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
			MRID: 1, MRIID: 5, MRURL: "https://gitlab.com/mr/5",
			SourceBranch: "dev", State: "closed", RepositoryID: 10,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		jsonStr := string(bytes)
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
					MRID: 1, MRIID: 1, MRURL: "https://example.com/mr/1",
					SourceBranch: "test", State: state, Action: action, RepositoryID: 1,
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
