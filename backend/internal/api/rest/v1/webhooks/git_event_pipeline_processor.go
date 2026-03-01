package webhooks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/ticket"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/eventbus"
)

// processPipelineEvent processes a pipeline webhook event
func (r *WebhookRouter) processPipelineEvent(ctx *WebhookContext) (map[string]interface{}, error) {
	// Extract pipeline data
	pipelineID := ctx.PipelineID
	pipelineStatus := ctx.PipelineStatus

	// Extract additional fields
	var pipelineURL, ref string
	if objAttrs, ok := ctx.Payload["object_attributes"].(map[string]interface{}); ok {
		if url, ok := objAttrs["url"].(string); ok {
			pipelineURL = url
		}
		if r, ok := objAttrs["ref"].(string); ok {
			ref = r
		}
	}

	r.logger.Info("processing pipeline event",
		"repo_id", ctx.RepoID,
		"pipeline_id", pipelineID,
		"status", pipelineStatus,
		"ref", ref)

	// Find and update associated MR
	mr, ticketID, podID := r.findAndUpdateMRForPipeline(ctx, pipelineID, pipelineStatus, pipelineURL, ref)

	// Publish event
	r.publishPipelineEvent(ctx, pipelineID, pipelineStatus, pipelineURL, ref, mr, ticketID, podID)

	return r.buildPipelineResult(pipelineID, pipelineStatus, ref, mr), nil
}

// findAndUpdateMRForPipeline finds an MR associated with the pipeline and updates its status
func (r *WebhookRouter) findAndUpdateMRForPipeline(ctx *WebhookContext, pipelineID int64, pipelineStatus, pipelineURL, ref string) (*ticket.MergeRequest, *int64, *int64) {
	if r.mrSyncService == nil {
		return nil, nil, nil
	}

	// Try to find MR by pipeline ID or branch name
	mr := r.findMRByPipeline(ctx.Context, ctx.OrganizationID, pipelineID, ref)
	if mr == nil {
		return nil, nil, nil
	}

	ticketID := mr.TicketID
	podID := mr.PodID

	// Update MR pipeline status
	mr.PipelineID = &pipelineID
	mr.PipelineStatus = &pipelineStatus
	if pipelineURL != "" {
		mr.PipelineURL = &pipelineURL
	}
	now := time.Now()
	mr.LastSyncedAt = &now
	r.db.WithContext(ctx.Context).Save(mr)

	return mr, ticketID, podID
}

// publishPipelineEvent publishes a pipeline event to the event bus
func (r *WebhookRouter) publishPipelineEvent(ctx *WebhookContext, pipelineID int64, pipelineStatus, pipelineURL, ref string, mr *ticket.MergeRequest, ticketID, podID *int64) {
	if r.eventBus == nil {
		return
	}

	pipelineEventData := &eventbus.PipelineEventData{
		PipelineID:     pipelineID,
		PipelineStatus: pipelineStatus,
		PipelineURL:    pipelineURL,
		SourceBranch:   ref,
		TicketID:       ticketID,
		PodID:          podID,
		RepositoryID:   ctx.RepoID,
	}
	if mr != nil {
		pipelineEventData.MRID = mr.ID
	}

	eventData, _ := json.Marshal(pipelineEventData)
	r.eventBus.Publish(ctx.Context, &eventbus.Event{
		Type:           eventbus.EventPipelineUpdated,
		Category:       eventbus.CategoryEntity,
		OrganizationID: ctx.OrganizationID,
		EntityType:     "pipeline",
		Data:           eventData,
		Timestamp:      time.Now().UnixMilli(),
	})
}

// buildPipelineResult builds the result map for a pipeline event
func (r *WebhookRouter) buildPipelineResult(pipelineID int64, pipelineStatus, ref string, mr *ticket.MergeRequest) map[string]interface{} {
	result := map[string]interface{}{
		"status":          "ok",
		"handler":         "pipeline",
		"pipeline_id":     pipelineID,
		"pipeline_status": pipelineStatus,
		"ref":             ref,
	}
	if mr != nil {
		result["mr_id"] = mr.ID
	}
	return result
}

// findMRByPipeline finds an MR by pipeline ID or branch name
func (r *WebhookRouter) findMRByPipeline(ctx context.Context, orgID, pipelineID int64, ref string) *ticket.MergeRequest {
	// First try by pipeline ID
	var mr ticket.MergeRequest
	if err := r.db.WithContext(ctx).
		Where("organization_id = ? AND pipeline_id = ?", orgID, pipelineID).
		First(&mr).Error; err == nil {
		return &mr
	}

	// Then try by branch name (ref)
	if ref != "" {
		if err := r.db.WithContext(ctx).
			Where("organization_id = ? AND source_branch = ? AND state != ?", orgID, ref, "merged").
			Order("created_at DESC").
			First(&mr).Error; err == nil {
			return &mr
		}
	}

	return nil
}
