package eventbus

import (
	"encoding/json"
	"testing"
)

// ===========================================
// PipelineEventData Tests
// ===========================================

func TestPipelineEventData_Serialization(t *testing.T) {
	t.Run("full PipelineEventData serialization", func(t *testing.T) {
		ticketID := int64(50)
		podID := int64(60)
		data := &PipelineEventData{
			MRID: 10, PipelineID: 12345, PipelineStatus: "success",
			PipelineURL:  "https://gitlab.com/org/repo/-/pipelines/12345",
			SourceBranch: "feature/new-feature",
			TicketID: &ticketID, PodID: &podID, RepositoryID: 300,
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
			PipelineID: 999, PipelineStatus: "failed", RepositoryID: 100,
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
			PipelineID: 100, PipelineStatus: "running", RepositoryID: 1,
		}

		bytes, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		jsonStr := string(bytes)
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
				PipelineID: 1, PipelineStatus: status, RepositoryID: 1,
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
		data := &PipelineEventData{
			PipelineID: 5000, PipelineStatus: "success",
			PipelineURL:  "https://gitlab.com/org/repo/-/pipelines/5000",
			SourceBranch: "main", RepositoryID: 10,
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
			MRID: 1, MRIID: 42, MRURL: "https://gitlab.com/mr/42",
			SourceBranch: "feature-branch", State: "opened", RepositoryID: 100,
		}
		mrDataBytes, _ := json.Marshal(mrData)

		event := &Event{
			Type: EventMRCreated, Category: CategoryEntity, OrganizationID: 1,
			EntityType: "merge_request", EntityID: "1",
			Data: mrDataBytes, Timestamp: 1234567890,
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
			PipelineID: 999, PipelineStatus: "success",
			PipelineURL: "https://gitlab.com/pipeline/999", RepositoryID: 50,
		}
		pipelineDataBytes, _ := json.Marshal(pipelineData)

		event := &Event{
			Type: EventPipelineUpdated, Category: CategoryEntity, OrganizationID: 2,
			EntityType: "pipeline", EntityID: "999",
			Data: pipelineDataBytes, Timestamp: 1234567890,
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
