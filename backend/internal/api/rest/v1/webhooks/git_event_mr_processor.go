package webhooks

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	ticketsvc "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

func (r *WebhookRouter) processMROrPipelineEvent(ctx *WebhookContext, objectKind string) (map[string]interface{}, error) {
	switch objectKind {
	case "merge_request":
		return r.processMergeRequestEvent(ctx)
	case "pipeline":
		return r.processPipelineEvent(ctx)
	default:
		return nil, fmt.Errorf("unsupported object kind: %s", objectKind)
	}
}

func (r *WebhookRouter) processMergeRequestEvent(ctx *WebhookContext) (map[string]interface{}, error) {
	mrData, action, err := r.extractMRData(ctx.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to extract MR data: %w", err)
	}

	r.logger.Info("processing MR event",
		"repo_id", ctx.RepoID,
		"mr_iid", mrData.IID,
		"action", action,
		"source_branch", mrData.SourceBranch,
		"state", mrData.State)

	podID, ticketID := r.findAssociatedPodAndTicket(ctx, mrData.SourceBranch)

	mr := r.createOrUpdateMRRecord(ctx, ticketID, mrData, podID)

	r.publishMREvent(ctx, mrData, action, mr, ticketID, podID)

	return r.buildMRResult(mrData, action, mr, ticketID, podID), nil
}

func (r *WebhookRouter) findAssociatedPodAndTicket(ctx *WebhookContext, sourceBranch string) (*int64, *int64) {
	var podID, ticketID *int64

	if r.podService != nil {
		pod, err := r.podService.FindByBranchAndRepo(ctx.Context, ctx.OrganizationID, ctx.RepoID, sourceBranch)
		if err == nil && pod != nil {
			podID = &pod.ID
			ticketID = pod.TicketID
		}
	}

	if ticketID == nil && r.mrSyncService != nil {
		t, err := r.mrSyncService.FindTicketByBranch(ctx.Context, ctx.OrganizationID, sourceBranch)
		if err == nil && t != nil {
			ticketID = &t.ID
		}
	}

	return podID, ticketID
}

func (r *WebhookRouter) createOrUpdateMRRecord(ctx *WebhookContext, ticketID *int64, mrData *ticketsvc.MRData, podID *int64) *ticket.MergeRequest {
	if r.db == nil {
		return nil
	}

	var existing ticket.MergeRequest
	err := r.db.WithContext(ctx.Context).
		Where("mr_url = ?", mrData.WebURL).
		First(&existing).Error

	now := time.Now()

	if err == nil {
		existing.Title = mrData.Title
		existing.State = mrData.State
		existing.PipelineStatus = mrData.PipelineStatus
		existing.PipelineID = mrData.PipelineID
		existing.PipelineURL = mrData.PipelineURL
		existing.MergeCommitSHA = mrData.MergeCommitSHA
		existing.MergedAt = mrData.MergedAt
		existing.LastSyncedAt = &now
		if podID != nil && existing.PodID == nil {
			existing.PodID = podID
		}
		if ticketID != nil && existing.TicketID == nil {
			existing.TicketID = ticketID
		}
		if err := r.db.WithContext(ctx.Context).Save(&existing).Error; err != nil {
			r.logger.Error("failed to update MR record", "error", err)
			return nil
		}
		return &existing
	}

	mr := &ticket.MergeRequest{
		OrganizationID: ctx.OrganizationID,
		RepositoryID:   ctx.RepoID,
		TicketID:       ticketID,
		PodID:          podID,
		MRIID:          mrData.IID,
		MRURL:          mrData.WebURL,
		SourceBranch:   mrData.SourceBranch,
		TargetBranch:   mrData.TargetBranch,
		Title:          mrData.Title,
		State:          mrData.State,
		PipelineStatus: mrData.PipelineStatus,
		PipelineID:     mrData.PipelineID,
		PipelineURL:    mrData.PipelineURL,
		MergeCommitSHA: mrData.MergeCommitSHA,
		MergedAt:       mrData.MergedAt,
		LastSyncedAt:   &now,
	}

	if err := r.db.WithContext(ctx.Context).Create(mr).Error; err != nil {
		r.logger.Error("failed to create MR record", "error", err)
		return nil
	}

	return mr
}

func (r *WebhookRouter) publishMREvent(ctx *WebhookContext, mrData *ticketsvc.MRData, action string, mr *ticket.MergeRequest, ticketID, podID *int64) {
	if r.eventBus == nil {
		return
	}

	eventType := r.determineMREventType(mrData.State, action)

	mrEventData := &eventbus.MREventData{
		MRIID:        mrData.IID,
		MRURL:        mrData.WebURL,
		SourceBranch: mrData.SourceBranch,
		TargetBranch: mrData.TargetBranch,
		Title:        mrData.Title,
		State:        mrData.State,
		Action:       action,
		TicketID:     ticketID,
		PodID:        podID,
		RepositoryID: ctx.RepoID,
	}
	if mr != nil {
		mrEventData.MRID = mr.ID
	}
	if mrData.PipelineStatus != nil {
		mrEventData.PipelineStatus = *mrData.PipelineStatus
	}

	eventData, _ := json.Marshal(mrEventData)
	r.eventBus.Publish(ctx.Context, &eventbus.Event{
		Type:           eventType,
		Category:       eventbus.CategoryEntity,
		OrganizationID: ctx.OrganizationID,
		EntityType:     "merge_request",
		Data:           eventData,
		Timestamp:      time.Now().UnixMilli(),
	})
}

func (r *WebhookRouter) buildMRResult(mrData *ticketsvc.MRData, action string, mr *ticket.MergeRequest, ticketID, podID *int64) map[string]interface{} {
	result := map[string]interface{}{
		"status":        "ok",
		"handler":       "merge_request",
		"mr_iid":        mrData.IID,
		"action":        action,
		"source_branch": mrData.SourceBranch,
		"state":         mrData.State,
		"ticket_id":     ticketID,
		"pod_id":        podID,
	}
	if mr != nil {
		result["mr_id"] = mr.ID
	}
	return result
}

func (r *WebhookRouter) extractMRData(payload map[string]interface{}) (*ticketsvc.MRData, string, error) {
	objAttrs, ok := payload["object_attributes"].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("missing object_attributes in MR webhook")
	}

	mrData := &ticketsvc.MRData{}

	if iid, ok := objAttrs["iid"].(float64); ok {
		mrData.IID = int(iid)
	}

	if url, ok := objAttrs["url"].(string); ok {
		mrData.WebURL = url
	}

	if title, ok := objAttrs["title"].(string); ok {
		mrData.Title = title
	}

	if sourceBranch, ok := objAttrs["source_branch"].(string); ok {
		mrData.SourceBranch = sourceBranch
	}
	if targetBranch, ok := objAttrs["target_branch"].(string); ok {
		mrData.TargetBranch = targetBranch
	}

	if state, ok := objAttrs["state"].(string); ok {
		mrData.State = state
	}

	action := ""
	if a, ok := objAttrs["action"].(string); ok {
		action = a
	}

	if pipeline, ok := objAttrs["head_pipeline"].(map[string]interface{}); ok {
		if status, ok := pipeline["status"].(string); ok {
			mrData.PipelineStatus = &status
		}
		if id, ok := pipeline["id"].(float64); ok {
			pipelineID := int64(id)
			mrData.PipelineID = &pipelineID
		}
		if url, ok := pipeline["web_url"].(string); ok {
			mrData.PipelineURL = &url
		}
	}

	if mergeCommitSHA, ok := objAttrs["merge_commit_sha"].(string); ok && mergeCommitSHA != "" {
		mrData.MergeCommitSHA = &mergeCommitSHA
	}

	if mergedAtStr, ok := objAttrs["merged_at"].(string); ok && mergedAtStr != "" {
		if mergedAt, err := time.Parse(time.RFC3339, mergedAtStr); err == nil {
			mrData.MergedAt = &mergedAt
		}
	}

	return mrData, action, nil
}

func (r *WebhookRouter) determineMREventType(state, action string) eventbus.EventType {
	switch {
	case action == "open" || action == "opened" || action == "reopen":
		return eventbus.EventMRCreated
	case state == "merged" || action == "merge":
		return eventbus.EventMRMerged
	case state == "closed" || action == "close":
		return eventbus.EventMRClosed
	default:
		return eventbus.EventMRUpdated
	}
}
