package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// SendTokenUsage sends a token usage report to the server (control message).
func (c *GRPCConnection) SendTokenUsage(podKey string, models []*runnerv1.TokenModelUsage) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_TokenUsage{
			TokenUsage: &runnerv1.TokenUsageReport{
				PodKey: podKey,
				Models: models,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}
