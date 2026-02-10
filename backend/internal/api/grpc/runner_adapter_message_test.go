package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== handleProtoMessage Tests ====================

func TestGRPCRunnerAdapter_HandleProtoMessage(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	t.Run("heartbeat message", func(t *testing.T) {
		var heartbeatReceived bool
		connMgr.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {
			heartbeatReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-node", data.NodeId)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Heartbeat{
				Heartbeat: &runnerv1.HeartbeatData{NodeId: "test-node"},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, heartbeatReceived)
	})

	t.Run("pod created message", func(t *testing.T) {
		var podCreatedReceived bool
		connMgr.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {
			podCreatedReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PodCreated{
				PodCreated: &runnerv1.PodCreatedEvent{PodKey: "test-pod"},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, podCreatedReceived)
	})

	t.Run("pod terminated message", func(t *testing.T) {
		var podTerminatedReceived bool
		connMgr.SetPodTerminatedCallback(func(runnerID int64, data *runnerv1.PodTerminatedEvent) {
			podTerminatedReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, int32(0), data.ExitCode)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PodTerminated{
				PodTerminated: &runnerv1.PodTerminatedEvent{PodKey: "test-pod", ExitCode: 0},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, podTerminatedReceived)
	})

	// NOTE: terminal output test removed - terminal output is exclusively streamed via Relay.
	// Runner no longer sends TerminalOutputEvent via gRPC.

	t.Run("agent status message", func(t *testing.T) {
		var agentStatusReceived bool
		connMgr.SetAgentStatusCallback(func(runnerID int64, data *runnerv1.AgentStatusEvent) {
			agentStatusReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, "running", data.Status)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_AgentStatus{
				AgentStatus: &runnerv1.AgentStatusEvent{PodKey: "test-pod", Status: "running"},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, agentStatusReceived)
	})

	t.Run("pty resized message", func(t *testing.T) {
		var ptyResizedReceived bool
		connMgr.SetPtyResizedCallback(func(runnerID int64, data *runnerv1.PtyResizedEvent) {
			ptyResizedReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, int32(120), data.Cols)
			assert.Equal(t, int32(40), data.Rows)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PtyResized{
				PtyResized: &runnerv1.PtyResizedEvent{PodKey: "test-pod", Cols: 120, Rows: 40},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, ptyResizedReceived)
	})

	t.Run("error message routes to HandlePodError callback", func(t *testing.T) {
		var podErrorReceived bool
		connMgr.SetPodErrorCallback(func(runnerID int64, data *runnerv1.ErrorEvent) {
			podErrorReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, "GIT_AUTH_FAILED", data.Code)
			assert.Equal(t, "authentication failed", data.Message)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Error{
				Error: &runnerv1.ErrorEvent{
					PodKey:  "test-pod",
					Code:    "GIT_AUTH_FAILED",
					Message: "authentication failed",
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, podErrorReceived, "HandlePodError callback should be invoked")
	})

	t.Run("initialize message", func(t *testing.T) {
		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialize{
				Initialize: &runnerv1.InitializeRequest{
					ProtocolVersion: 2,
				},
			},
		}
		// Should send InitializeResult
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)

		// Drain the response
		select {
		case <-conn.Send:
			// Expected
		default:
			// May have been consumed already
		}
	})

	t.Run("initialized message", func(t *testing.T) {
		var initCallbackCalled bool
		connMgr.SetInitializedCallback(func(runnerID int64, availableAgents []string) {
			initCallbackCalled = true
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialized{
				Initialized: &runnerv1.InitializedConfirm{
					AvailableAgents: []string{"test-agent"},
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, initCallbackCalled)
	})

	t.Run("unknown message type", func(t *testing.T) {
		// This should log a warning but not panic
		msg := &runnerv1.RunnerMessage{
			Payload: nil, // Unknown/nil payload
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
	})

	t.Run("osc notification message", func(t *testing.T) {
		var oscNotificationReceived bool
		connMgr.SetOSCNotificationCallback(func(runnerID int64, data *runnerv1.OSCNotificationEvent) {
			oscNotificationReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, "Build Complete", data.Title)
			assert.Equal(t, "Project compiled successfully", data.Body)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_OscNotification{
				OscNotification: &runnerv1.OSCNotificationEvent{
					PodKey:    "test-pod",
					Title:     "Build Complete",
					Body:      "Project compiled successfully",
					Timestamp: 1234567890,
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, oscNotificationReceived)
	})

	t.Run("osc title message", func(t *testing.T) {
		var oscTitleReceived bool
		connMgr.SetOSCTitleCallback(func(runnerID int64, data *runnerv1.OSCTitleEvent) {
			oscTitleReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, "My Terminal Title", data.Title)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_OscTitle{
				OscTitle: &runnerv1.OSCTitleEvent{
					PodKey: "test-pod",
					Title:  "My Terminal Title",
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, oscTitleReceived)
	})

	t.Run("sandboxes status message", func(t *testing.T) {
		var sandboxesStatusReceived bool
		connMgr.SetSandboxesStatusCallback(func(runnerID int64, data *runnerv1.SandboxesStatusEvent) {
			sandboxesStatusReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "req-123", data.RequestId)
			assert.Len(t, data.Sandboxes, 1)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_SandboxesStatus{
				SandboxesStatus: &runnerv1.SandboxesStatusEvent{
					RequestId: "req-123",
					Sandboxes: []*runnerv1.SandboxStatus{
						{PodKey: "pod-1", Exists: true},
					},
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, sandboxesStatusReceived)
	})

	t.Run("pod init progress message", func(t *testing.T) {
		var podInitProgressReceived bool
		connMgr.SetPodInitProgressCallback(func(runnerID int64, data *runnerv1.PodInitProgressEvent) {
			podInitProgressReceived = true
			assert.Equal(t, int64(1), runnerID)
			assert.Equal(t, "test-pod", data.PodKey)
			assert.Equal(t, "cloning", data.Phase)
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PodInitProgress{
				PodInitProgress: &runnerv1.PodInitProgressEvent{
					PodKey: "test-pod",
					Phase:  "cloning",
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		assert.True(t, podInitProgressReceived)
	})
}
