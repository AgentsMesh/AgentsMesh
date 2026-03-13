package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// publishRunEvent publishes a loop run event to the event bus
func (o *LoopOrchestrator) publishRunEvent(orgID int64, eventType eventbus.EventType, run *loopDomain.LoopRun) {
	if o.eventBus == nil {
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"loop_id":    run.LoopID,
		"run_id":     run.ID,
		"run_number": run.RunNumber,
		"status":     run.Status,
		"pod_key":    run.PodKey,
	})

	_ = o.eventBus.Publish(context.Background(), &eventbus.Event{
		Type:           eventType,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "loop_run",
		EntityID:       fmt.Sprintf("%d", run.ID),
		Data:           data,
		Timestamp:      time.Now().UnixMilli(),
	})
}

// publishWarningEvent publishes a loop run warning event to the event bus
func (o *LoopOrchestrator) publishWarningEvent(orgID int64, loopID int64, runID int64, runNumber int, warning string, detail string) {
	if o.eventBus == nil {
		return
	}

	data, _ := json.Marshal(eventbus.LoopRunWarningData{
		LoopID:    loopID,
		RunID:     runID,
		RunNumber: runNumber,
		Warning:   warning,
		Detail:    detail,
	})

	_ = o.eventBus.Publish(context.Background(), &eventbus.Event{
		Type:           eventbus.EventLoopRunWarning,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "loop_run",
		EntityID:       fmt.Sprintf("%d", runID),
		Data:           data,
		Timestamp:      time.Now().UnixMilli(),
	})
}

// sendWebhookCallback POSTs run result to the loop's callback URL.
// Fire-and-forget: errors are logged but do not affect the run.
func (o *LoopOrchestrator) sendWebhookCallback(callbackURL string, loop *loopDomain.Loop, run *loopDomain.LoopRun, status string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"loop_id":      loop.ID,
		"loop_slug":    loop.Slug,
		"loop_name":    loop.Name,
		"run_id":       run.ID,
		"run_number":   run.RunNumber,
		"status":       status,
		"trigger":      run.TriggerType,
		"exit_summary": run.ExitSummary,
		"started_at": func() string {
			if run.StartedAt != nil {
				return run.StartedAt.Format(time.RFC3339)
			}
			return ""
		}(),
		"finished_at": func() string {
			if run.FinishedAt != nil {
				return run.FinishedAt.Format(time.RFC3339)
			}
			return time.Now().Format(time.RFC3339)
		}(),
	})

	resp, err := o.httpClient.Post(callbackURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		o.logger.Warn("webhook callback failed",
			"loop_id", loop.ID, "run_id", run.ID, "url", callbackURL, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		o.logger.Warn("webhook callback returned error",
			"loop_id", loop.ID, "run_id", run.ID, "url", callbackURL, "status", resp.StatusCode)
	}
}

// postTicketComment creates a comment on the associated ticket with run results.
func (o *LoopOrchestrator) postTicketComment(ctx context.Context, ticketID int64, userID int64, loop *loopDomain.Loop, run *loopDomain.LoopRun, status string) {
	statusEmoji := "✅"
	switch status {
	case loopDomain.RunStatusFailed:
		statusEmoji = "❌"
	case loopDomain.RunStatusTimeout:
		statusEmoji = "⏰"
	case loopDomain.RunStatusCancelled:
		statusEmoji = "⊘"
	}

	durationStr := "-"
	if run.StartedAt != nil && run.FinishedAt != nil {
		durationStr = fmt.Sprintf("%.0fs", run.FinishedAt.Sub(*run.StartedAt).Seconds())
	}

	content := fmt.Sprintf(
		"%s **Loop Run #%d** — %s\n\nLoop: **%s** (`%s`)\nDuration: %s\nTrigger: %s",
		statusEmoji, run.RunNumber, status, loop.Name, loop.Slug, durationStr, run.TriggerType,
	)

	if _, err := o.ticketService.CreateComment(ctx, ticketID, userID, content, nil, nil); err != nil {
		o.logger.Warn("failed to create ticket comment for loop run",
			"loop_id", loop.ID, "run_id", run.ID, "ticket_id", ticketID, "error", err)
	}
}

// resolvePrompt merges default variables with trigger-time overrides,
// then substitutes {{key}} placeholders in the template.
func resolvePrompt(template string, defaults json.RawMessage, overrides json.RawMessage) string {
	vars := make(map[string]interface{})
	if len(defaults) > 0 {
		_ = json.Unmarshal(defaults, &vars)
	}
	if len(overrides) > 0 {
		var ov map[string]interface{}
		if err := json.Unmarshal(overrides, &ov); err == nil {
			for k, v := range ov {
				vars[k] = v
			}
		}
	}

	result := template
	for k, v := range vars {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
	}
	return result
}
