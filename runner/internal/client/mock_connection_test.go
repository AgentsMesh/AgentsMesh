package client

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockConnection_Lifecycle(t *testing.T) {
	mc := NewMockConnection()

	assert.False(t, mc.IsStarted())
	assert.False(t, mc.IsStopped())

	mc.Start()
	assert.True(t, mc.IsStarted())

	mc.Stop()
	assert.True(t, mc.IsStopped())
}

func TestMockConnection_Connect(t *testing.T) {
	mc := NewMockConnection()
	assert.NoError(t, mc.Connect())
}

func TestMockConnection_SetHandler(t *testing.T) {
	mc := NewMockConnection()
	handler := &mockHandler{pods: []PodInfo{{PodKey: "p1"}}}
	mc.SetHandler(handler)

	pods := mc.GetPods()
	require.Len(t, pods, 1)
	assert.Equal(t, "p1", pods[0].PodKey)
}

func TestMockConnection_GetPods_NilHandler(t *testing.T) {
	mc := NewMockConnection()
	assert.Nil(t, mc.GetPods())
}

func TestMockConnection_SendEvents(t *testing.T) {
	mc := NewMockConnection()

	_ = mc.SendPodCreated("pod-1", 1234, "/sandbox", "main")
	_ = mc.SendPodTerminated("pod-1", 0, "", "completed")
	_ = mc.SendError("pod-1", "err", "msg")
	_ = mc.SendPodInitProgress("pod-1", "clone", 50, "cloning...")
	_ = mc.SendRequestRelayToken("pod-1", "wss://relay")
	_ = mc.SendSandboxesStatus("req-1", nil)
	_ = mc.SendObservePodResult("req-1", "pod-1", "output", "", 0, 0, 1, false, "")
	_ = mc.SendOSCNotification("pod-1", "title", "body")
	_ = mc.SendOSCTitle("pod-1", "vim")
	_ = mc.SendAgentStatus("pod-1", "executing")
	_ = mc.SendMessage(&runnerv1.RunnerMessage{})

	events := mc.GetEvents()
	assert.Len(t, events, 11)
}

func TestMockConnection_SendErr(t *testing.T) {
	mc := NewMockConnection()
	mc.SendErr = assert.AnError

	assert.Error(t, mc.SendPodCreated("pod-1", 1, "", ""))
	assert.Error(t, mc.SendPodTerminated("pod-1", 0, "", "completed"))
	assert.Error(t, mc.SendError("pod-1", "", ""))
	assert.Error(t, mc.SendAgentStatus("pod-1", ""))
	assert.Error(t, mc.SendMessage(&runnerv1.RunnerMessage{}))
}

func TestMockConnection_QueueLength(t *testing.T) {
	mc := NewMockConnection()
	assert.Equal(t, 0, mc.QueueLength())

	_ = mc.SendPodCreated("pod-1", 1, "", "")
	assert.Equal(t, 1, mc.QueueLength())
}

func TestMockConnection_QueueCapacity(t *testing.T) {
	mc := NewMockConnection()
	assert.Equal(t, 100, mc.QueueCapacity())
}

func TestMockConnection_QueueUsage(t *testing.T) {
	mc := NewMockConnection()
	assert.Equal(t, 0.0, mc.QueueUsage())
}

func TestMockConnection_Reset(t *testing.T) {
	mc := NewMockConnection()
	mc.Start()
	_ = mc.SendPodCreated("pod-1", 1, "", "")

	mc.Reset()
	assert.False(t, mc.IsStarted())
	assert.False(t, mc.IsStopped())
	assert.Empty(t, mc.GetEvents())
}

func TestMockConnection_OrgSlug(t *testing.T) {
	mc := NewMockConnection()
	mc.SetOrgSlug("org-1") // no-op
	assert.Equal(t, "", mc.GetOrgSlug())
}

func TestMockConnection_SimulateCreatePod(t *testing.T) {
	mc := NewMockConnection()
	handler := &mockHandler{}
	mc.SetHandler(handler)

	cmd := &runnerv1.CreatePodCommand{PodKey: "pod-1"}
	err := mc.SimulateCreatePod(cmd)
	assert.NoError(t, err)

	handler.mu.Lock()
	assert.True(t, handler.createPodCalled)
	handler.mu.Unlock()
}

func TestMockConnection_SimulateTerminatePod(t *testing.T) {
	mc := NewMockConnection()
	handler := &mockHandler{}
	mc.SetHandler(handler)

	err := mc.SimulateTerminatePod(TerminatePodRequest{PodKey: "pod-1"})
	assert.NoError(t, err)

	handler.mu.Lock()
	assert.True(t, handler.terminatePodCalled)
	handler.mu.Unlock()
}

func TestMockConnection_SimulatePodInput(t *testing.T) {
	mc := NewMockConnection()
	handler := &mockHandler{}
	mc.SetHandler(handler)

	err := mc.SimulatePodInput(PodInputRequest{PodKey: "pod-1", Data: []byte("ls")})
	assert.NoError(t, err)

	handler.mu.Lock()
	assert.True(t, handler.terminalInputCalled)
	handler.mu.Unlock()
}

func TestMockConnection_SimulateNilHandler(t *testing.T) {
	mc := NewMockConnection()
	// No handler set — all simulate methods should return nil

	assert.NoError(t, mc.SimulateCreatePod(&runnerv1.CreatePodCommand{}))
	assert.NoError(t, mc.SimulateTerminatePod(TerminatePodRequest{}))
	assert.NoError(t, mc.SimulatePodInput(PodInputRequest{}))
	assert.NoError(t, mc.SimulateSubscribePod(SubscribePodRequest{}))
	assert.NoError(t, mc.SimulateUnsubscribePod(UnsubscribePodRequest{}))
	assert.NoError(t, mc.SimulateQuerySandboxes(QuerySandboxesRequest{}))
	assert.NoError(t, mc.SimulateCreateAutopilot(&runnerv1.CreateAutopilotCommand{}))
	assert.NoError(t, mc.SimulateAutopilotControl(&runnerv1.AutopilotControlCommand{}))
}
