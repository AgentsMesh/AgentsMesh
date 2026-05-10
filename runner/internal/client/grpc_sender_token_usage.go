package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// SendTokenUsage sends a token usage report to the server (control message).
// podStartedAt enables backend-side detection of long-running sessions that
// produced empty token reports (parser regression smell). A zero time is
// explicitly serialized as 0 (not the year-1 default of t.Unix() which is
// -62135596800) so the backend's "legacy / no pod_started_at" branch fires.
func (c *GRPCConnection) SendTokenUsage(podKey string, models []*runnerv1.TokenModelUsage, podStartedAt time.Time) error {
	var startedAtSeconds int64
	if !podStartedAt.IsZero() {
		startedAtSeconds = podStartedAt.Unix()
	}
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_TokenUsage{
			TokenUsage: &runnerv1.TokenUsageReport{
				PodKey:                  podKey,
				Models:                  models,
				PodStartedAtUnixSeconds: startedAtSeconds,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}
