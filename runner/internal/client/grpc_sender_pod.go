// Package client provides gRPC connection management for Runner.
package client

import (
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// SendPodCreated sends a pod_created event to the server (control message).
func (c *GRPCConnection) SendPodCreated(podKey string, pid int32, sandboxPath, branchName string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodCreated{
			PodCreated: &runnerv1.PodCreatedEvent{
				PodKey:      podKey,
				Pid:         pid,
				SandboxPath: sandboxPath,
				BranchName:  branchName,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendPodTerminated sends a pod_terminated event to the server (control message).
func (c *GRPCConnection) SendPodTerminated(podKey string, exitCode int32, errorMsg string, status string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodTerminated{
			PodTerminated: &runnerv1.PodTerminatedEvent{
				PodKey:       podKey,
				ExitCode:     exitCode,
				ErrorMessage: errorMsg,
				Status:       status,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}

// SendPodInitProgress sends a pod initialization progress event to the server (control message).
func (c *GRPCConnection) SendPodInitProgress(podKey, phase string, progress int32, message string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodInitProgress{
			PodInitProgress: &runnerv1.PodInitProgressEvent{
				PodKey:   podKey,
				Phase:    phase,
				Progress: progress,
				Message:  message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.sendControl(msg)
}
