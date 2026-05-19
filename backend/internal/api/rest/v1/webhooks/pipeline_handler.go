package webhooks

import (
	"fmt"
	"log/slog"
)

const (
	PipelineStatusPending  = "pending"
	PipelineStatusRunning  = "running"
	PipelineStatusSuccess  = "success"
	PipelineStatusFailed   = "failed"
	PipelineStatusCanceled = "canceled"
	PipelineStatusSkipped  = "skipped"
	PipelineStatusManual   = "manual"
)

type PipelineHandler struct {
	logger *slog.Logger
}

func NewPipelineHandler(logger *slog.Logger) *PipelineHandler {
	return &PipelineHandler{logger: logger}
}

func (h *PipelineHandler) CanHandle(ctx *WebhookContext) bool {
	return ctx.ObjectKind == "pipeline" && ctx.PipelineID > 0
}

func (h *PipelineHandler) Handle(ctx *WebhookContext) (map[string]interface{}, error) {
	h.logger.Info("processing pipeline event",
		"project_id", ctx.ProjectID,
		"pipeline_id", ctx.PipelineID,
		"status", ctx.PipelineStatus)

	var pipelineURL string
	if objAttrs, ok := ctx.Payload["object_attributes"].(map[string]interface{}); ok {
		if url, ok := objAttrs["url"].(string); ok {
			pipelineURL = url
		}
	}

	result := map[string]interface{}{
		"status":          "ok",
		"pipeline_id":     ctx.PipelineID,
		"pipeline_status": ctx.PipelineStatus,
		"pipeline_url":    pipelineURL,
	}

	return result, nil
}

type MergeRequestHandler struct {
	logger *slog.Logger
}

func NewMergeRequestHandler(logger *slog.Logger) *MergeRequestHandler {
	return &MergeRequestHandler{logger: logger}
}

func (h *MergeRequestHandler) CanHandle(ctx *WebhookContext) bool {
	if ctx.ObjectKind != "merge_request" {
		return false
	}

	if objAttrs, ok := ctx.Payload["object_attributes"].(map[string]interface{}); ok {
		if sourceBranch, ok := objAttrs["source_branch"].(string); ok && sourceBranch != "" {
			return true
		}
	}

	return false
}

func (h *MergeRequestHandler) Handle(ctx *WebhookContext) (map[string]interface{}, error) {
	objAttrs, ok := ctx.Payload["object_attributes"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing object_attributes in MR webhook")
	}

	sourceBranch := objAttrs["source_branch"].(string)
	action := ""
	if a, ok := objAttrs["action"].(string); ok {
		action = a
	}

	h.logger.Info("processing MR event",
		"mr_iid", ctx.MRIID,
		"action", action,
		"source_branch", sourceBranch)

	result := map[string]interface{}{
		"status":        "ok",
		"mr_iid":        ctx.MRIID,
		"action":        action,
		"source_branch": sourceBranch,
	}

	if title, ok := objAttrs["title"].(string); ok {
		result["title"] = title
	}
	if state, ok := objAttrs["state"].(string); ok {
		result["state"] = state
	}
	if targetBranch, ok := objAttrs["target_branch"].(string); ok {
		result["target_branch"] = targetBranch
	}
	if url, ok := objAttrs["url"].(string); ok {
		result["mr_url"] = url
	}

	return result, nil
}

type PushHandler struct {
	logger *slog.Logger
}

func NewPushHandler(logger *slog.Logger) *PushHandler {
	return &PushHandler{logger: logger}
}

func (h *PushHandler) CanHandle(ctx *WebhookContext) bool {
	return ctx.ObjectKind == "push"
}

func (h *PushHandler) Handle(ctx *WebhookContext) (map[string]interface{}, error) {
	var ref, before, after string
	var totalCommits int

	if r, ok := ctx.Payload["ref"].(string); ok {
		ref = r
	}
	if b, ok := ctx.Payload["before"].(string); ok {
		before = b
	}
	if a, ok := ctx.Payload["after"].(string); ok {
		after = a
	}
	if commits, ok := ctx.Payload["commits"].([]interface{}); ok {
		totalCommits = len(commits)
	}

	h.logger.Info("processing push event",
		"project_id", ctx.ProjectID,
		"ref", ref,
		"commits", totalCommits)

	result := map[string]interface{}{
		"status":        "ok",
		"ref":           ref,
		"before":        before,
		"after":         after,
		"total_commits": totalCommits,
	}

	return result, nil
}

func SetupDefaultHandlers(registry *HandlerRegistry, logger *slog.Logger) {
	registry.Register("pipeline", NewPipelineHandler(logger))

	registry.Register("merge_request", NewMergeRequestHandler(logger))

	pushHandler := NewCompositeHandler(logger)
	pushHandler.AddSubHandler(NewPushHandler(logger))
	registry.Register("push", pushHandler)
}
