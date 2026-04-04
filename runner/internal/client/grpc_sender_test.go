package client

import (
	"context"
	"io"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// fakeStream implements grpc.BidiStreamingClient for testing.
type fakeStream struct {
	grpc.ClientStream
	sentMsgs []*runnerv1.RunnerMessage
}

func (f *fakeStream) Send(msg *runnerv1.RunnerMessage) error {
	f.sentMsgs = append(f.sentMsgs, msg)
	return nil
}

func (f *fakeStream) Recv() (*runnerv1.ServerMessage, error) {
	return nil, io.EOF
}

func (f *fakeStream) CloseSend() error             { return nil }
func (f *fakeStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeStream) Trailer() metadata.MD         { return nil }
func (f *fakeStream) Context() context.Context     { return context.Background() }
func (f *fakeStream) SendMsg(m interface{}) error  { return nil }
func (f *fakeStream) RecvMsg(m interface{}) error  { return nil }

// setFakeStream sets a non-nil stream on the connection for testing.
func setFakeStream(conn *GRPCConnection) {
	conn.stream = &fakeStream{}
}

func TestSendControl_Success(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  "test",
				Code:    "test_error",
				Message: "test message",
			},
		},
	}

	err := conn.sendControl(msg)
	require.NoError(t, err)

	select {
	case received := <-conn.controlCh:
		assert.NotNil(t, received)
	default:
		t.Fatal("no message in control channel")
	}
}

func TestSendControl_NoStream(t *testing.T) {
	conn := newTestConnection()
	// stream is nil

	err := conn.sendControl(&runnerv1.RunnerMessage{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stream not connected")
}

func TestSendControl_Stopped(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	close(conn.stopCh)

	// Fill the control channel to force the select to fall through to stopCh
	for i := 0; i < cap(conn.controlCh); i++ {
		conn.controlCh <- &runnerv1.RunnerMessage{}
	}

	err := conn.sendControl(&runnerv1.RunnerMessage{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection stopped")
}

func TestSendControl_BufferFull(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	// Fill the channel
	for i := 0; i < cap(conn.controlCh); i++ {
		conn.controlCh <- &runnerv1.RunnerMessage{}
	}

	err := conn.sendControl(&runnerv1.RunnerMessage{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "control buffer full")
}

func TestSendTerminal_BeforeInit(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initialized = false

	// Should silently drop (return nil, not error)
	err := conn.sendTerminal(&runnerv1.RunnerMessage{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(conn.terminalCh))
}

func TestSendTerminal_AfterInit(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initialized = true

	err := conn.sendTerminal(&runnerv1.RunnerMessage{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(conn.terminalCh))
}

func TestSendTerminal_NoStream(t *testing.T) {
	conn := newTestConnection()
	conn.initialized = true
	// stream is nil

	err := conn.sendTerminal(&runnerv1.RunnerMessage{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stream not connected")
}

func TestSendTerminal_BufferFull(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initialized = true

	for i := 0; i < cap(conn.terminalCh); i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}

	// Silent drop for terminal messages
	err := conn.sendTerminal(&runnerv1.RunnerMessage{})
	assert.NoError(t, err)
}

func TestSendPodCreated(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendPodCreated("pod-1", 1234, "/sandbox", "main")
	require.NoError(t, err)

	msg := <-conn.controlCh
	created, ok := msg.Payload.(*runnerv1.RunnerMessage_PodCreated)
	require.True(t, ok)
	assert.Equal(t, "pod-1", created.PodCreated.PodKey)
	assert.Equal(t, int32(1234), created.PodCreated.Pid)
	assert.Equal(t, "/sandbox", created.PodCreated.SandboxPath)
	assert.Equal(t, "main", created.PodCreated.BranchName)
}

func TestSendPodTerminated(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendPodTerminated("pod-1", 0, "", "completed")
	require.NoError(t, err)

	msg := <-conn.controlCh
	terminated, ok := msg.Payload.(*runnerv1.RunnerMessage_PodTerminated)
	require.True(t, ok)
	assert.Equal(t, "pod-1", terminated.PodTerminated.PodKey)
}

func TestSendError(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendError("pod-1", "test_code", "test message")
	require.NoError(t, err)

	msg := <-conn.controlCh
	errEvt, ok := msg.Payload.(*runnerv1.RunnerMessage_Error)
	require.True(t, ok)
	assert.Equal(t, "test_code", errEvt.Error.Code)
}

func TestSendPodInitProgress(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendPodInitProgress("pod-1", "cloning", 50, "Cloning...")
	require.NoError(t, err)

	msg := <-conn.controlCh
	progress, ok := msg.Payload.(*runnerv1.RunnerMessage_PodInitProgress)
	require.True(t, ok)
	assert.Equal(t, "cloning", progress.PodInitProgress.Phase)
	assert.Equal(t, int32(50), progress.PodInitProgress.Progress)
}

func TestSendAgentStatus(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.initialized = true

	err := conn.SendAgentStatus("pod-1", "executing")
	require.NoError(t, err)

	msg := <-conn.terminalCh
	status, ok := msg.Payload.(*runnerv1.RunnerMessage_AgentStatus)
	require.True(t, ok)
	assert.Equal(t, "executing", status.AgentStatus.Status)
}

func TestSendPodResized(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendPodResized("pod-1", 120, 40)
	require.NoError(t, err)

	msg := <-conn.controlCh
	resized, ok := msg.Payload.(*runnerv1.RunnerMessage_PodResized)
	require.True(t, ok)
	assert.Equal(t, int32(120), resized.PodResized.Cols)
}

func TestSendOSCNotification(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendOSCNotification("pod-1", "Build Done", "Tests passed")
	require.NoError(t, err)

	msg := <-conn.controlCh
	osc, ok := msg.Payload.(*runnerv1.RunnerMessage_OscNotification)
	require.True(t, ok)
	assert.Equal(t, "Build Done", osc.OscNotification.Title)
}

func TestSendOSCTitle(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendOSCTitle("pod-1", "vim main.go")
	require.NoError(t, err)

	msg := <-conn.controlCh
	osc, ok := msg.Payload.(*runnerv1.RunnerMessage_OscTitle)
	require.True(t, ok)
	assert.Equal(t, "vim main.go", osc.OscTitle.Title)
}

func TestSendRequestRelayToken(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	err := conn.SendRequestRelayToken("pod-1", "wss://relay.example.com")
	require.NoError(t, err)

	msg := <-conn.controlCh
	relay, ok := msg.Payload.(*runnerv1.RunnerMessage_RequestRelayToken)
	require.True(t, ok)
	assert.Equal(t, "wss://relay.example.com", relay.RequestRelayToken.RelayUrl)
}

func TestSendSandboxesStatus(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	results := []*SandboxStatusInfo{
		{PodKey: "pod-1", Exists: true, SandboxPath: "/sandbox/pod-1"},
		{PodKey: "pod-2", Exists: false},
	}

	err := conn.SendSandboxesStatus("req-123", results)
	require.NoError(t, err)

	msg := <-conn.controlCh
	status, ok := msg.Payload.(*runnerv1.RunnerMessage_SandboxesStatus)
	require.True(t, ok)
	assert.Equal(t, "req-123", status.SandboxesStatus.RequestId)
	assert.Len(t, status.SandboxesStatus.Sandboxes, 2)
}

func TestSendMessage_SetsTimestamp(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{PodKey: "test"},
		},
	}

	err := conn.SendMessage(msg)
	require.NoError(t, err)

	received := <-conn.controlCh
	assert.NotZero(t, received.Timestamp)
}

func TestQueueLength(t *testing.T) {
	conn := newTestConnection()
	assert.Equal(t, 0, conn.QueueLength())
	conn.terminalCh <- &runnerv1.RunnerMessage{}
	assert.Equal(t, 1, conn.QueueLength())
}

func TestQueueCapacity(t *testing.T) {
	conn := newTestConnection()
	assert.Equal(t, 100, conn.QueueCapacity())
}

func TestQueueUsage(t *testing.T) {
	conn := newTestConnection()
	assert.Equal(t, 0.0, conn.QueueUsage())
	for i := 0; i < 50; i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}
	assert.InDelta(t, 0.5, conn.QueueUsage(), 0.01)
}

func TestDrainTerminalQueue(t *testing.T) {
	conn := newTestConnection()
	for i := 0; i < 10; i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}
	conn.drainTerminalQueue()
	assert.Equal(t, 0, len(conn.terminalCh))
}

func TestDrainTerminalQueue_Empty(t *testing.T) {
	conn := newTestConnection()
	conn.drainTerminalQueue()
	assert.Equal(t, 0, len(conn.terminalCh))
}

// sendError (internal helper) should not return error to caller
func TestSendErrorInternal(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	// Should not panic even when stream is available
	conn.sendError("pod-1", "code", "message")

	msg := <-conn.controlCh
	assert.NotNil(t, msg)
}

func TestSendErrorInternal_NoStream(t *testing.T) {
	conn := newTestConnection()
	// stream is nil, should not panic
	conn.sendError("pod-1", "code", "message")
}

func TestLastActivityTime(t *testing.T) {
	conn := newTestConnection()

	// Initially zero
	assert.True(t, conn.LastActivityTime().IsZero())

	// Set send time
	now := time.Now()
	conn.lastSendTime.Store(now.UnixNano())
	assert.False(t, conn.LastActivityTime().IsZero())

	// Set recv time later
	later := now.Add(time.Second)
	conn.lastRecvTime.Store(later.UnixNano())
	assert.True(t, conn.LastActivityTime().After(now))
}
