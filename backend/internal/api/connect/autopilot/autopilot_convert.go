package autopilotconnect

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	apv1 "github.com/anthropics/agentsmesh/proto/gen/go/autopilot/v1"
)

func toProtoController(c *agentpod.AutopilotController) *apv1.AutopilotController {
	if c == nil {
		return nil
	}
	out := &apv1.AutopilotController{
		Id:                     c.ID,
		AutopilotControllerKey: c.AutopilotControllerKey,
		PodKey:                 c.PodKey,
		Phase:                  c.Phase,
		CurrentIteration:       c.CurrentIteration,
		MaxIterations:          c.MaxIterations,
		CircuitBreaker: &apv1.CircuitBreaker{
			State: c.CircuitBreakerState,
		},
		UserTakeover: c.UserTakeover,
		Prompt:       c.Prompt,
		CreatedAt:    c.CreatedAt.UTC().Format(time.RFC3339),
	}
	if c.CircuitBreakerReason != nil {
		out.CircuitBreaker.Reason = *c.CircuitBreakerReason
	}
	if c.StartedAt != nil {
		v := c.StartedAt.UTC().Format(time.RFC3339)
		out.StartedAt = &v
	}
	if c.LastIterationAt != nil {
		v := c.LastIterationAt.UTC().Format(time.RFC3339)
		out.LastIterationAt = &v
	}
	return out
}

func toProtoIteration(it *agentpod.AutopilotIteration) *apv1.AutopilotIteration {
	if it == nil {
		return nil
	}
	out := &apv1.AutopilotIteration{
		Id:              it.ID,
		IterationNumber: int64(it.Iteration),
		Status:          it.Phase,
	}
	if it.Summary != nil {
		out.Result = *it.Summary
	}
	// AutopilotIteration has CreatedAt only; render it as started_at to
	// match the renderer-facing shape.
	v := it.CreatedAt.UTC().Format(time.RFC3339)
	out.StartedAt = &v
	return out
}
