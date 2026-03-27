package v1

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/service/geo"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// --- mock types used only in relay connect tests ---
// These are separate from pod_acp_test.go mocks to avoid name collision.

// mockRelayPodService implements PodServiceForHandler for relay connect tests.
type mockRelayPodService struct {
	getPodFn func(ctx context.Context, podKey string) (*agentpod.Pod, error)
}

func (m *mockRelayPodService) ListPods(context.Context, int64, []string, int64, int, int) ([]*agentpod.Pod, int64, error) {
	return nil, 0, nil
}
func (m *mockRelayPodService) CreatePod(context.Context, *agentpodSvc.CreatePodRequest) (*agentpod.Pod, error) {
	return nil, nil
}
func (m *mockRelayPodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	if m.getPodFn != nil {
		return m.getPodFn(ctx, podKey)
	}
	return nil, errors.New("not found")
}
func (m *mockRelayPodService) TerminatePod(context.Context, string) error { return nil }
func (m *mockRelayPodService) GetPodsByTicket(context.Context, int64) ([]*agentpod.Pod, error) {
	return nil, nil
}
func (m *mockRelayPodService) UpdateAlias(context.Context, string, *string) error { return nil }
func (m *mockRelayPodService) GetActivePodBySourcePodKey(context.Context, string) (*agentpod.Pod, error) {
	return nil, nil
}

// mockRelayCommandSender implements runner.RunnerCommandSender for relay connect tests (no-op).
type mockRelayCommandSender struct{}

func (m *mockRelayCommandSender) SendCreatePod(context.Context, int64, *runnerv1.CreatePodCommand) error {
	return nil
}
func (m *mockRelayCommandSender) SendTerminatePod(context.Context, int64, string) error { return nil }
func (m *mockRelayCommandSender) SendPodInput(context.Context, int64, string, []byte) error {
	return nil
}
func (m *mockRelayCommandSender) SendPrompt(context.Context, int64, string, string) error {
	return nil
}
func (m *mockRelayCommandSender) SendSubscribePod(context.Context, int64, string, string, string, bool, int32) error {
	return nil
}
func (m *mockRelayCommandSender) SendUnsubscribePod(context.Context, int64, string) error {
	return nil
}
func (m *mockRelayCommandSender) SendObservePod(context.Context, int64, string, string, int32, bool) error {
	return nil
}
func (m *mockRelayCommandSender) SendCreateAutopilot(int64, *runnerv1.CreateAutopilotCommand) error {
	return nil
}
func (m *mockRelayCommandSender) SendAutopilotControl(int64, *runnerv1.AutopilotControlCommand) error {
	return nil
}

// mockRelayCommandSenderConfigurable allows configuring individual method behaviors.
type mockRelayCommandSenderConfigurable struct {
	sendSubscribePodFn func(context.Context, int64, string, string, string, bool, int32) error
}

func (m *mockRelayCommandSenderConfigurable) SendCreatePod(context.Context, int64, *runnerv1.CreatePodCommand) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendTerminatePod(context.Context, int64, string) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendPodInput(context.Context, int64, string, []byte) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendPrompt(context.Context, int64, string, string) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendSubscribePod(ctx context.Context, runnerID int64, podKey, relayURL, token string, snapshot bool, lines int32) error {
	if m.sendSubscribePodFn != nil {
		return m.sendSubscribePodFn(ctx, runnerID, podKey, relayURL, token, snapshot, lines)
	}
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendUnsubscribePod(context.Context, int64, string) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendObservePod(context.Context, int64, string, string, int32, bool) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendCreateAutopilot(int64, *runnerv1.CreateAutopilotCommand) error {
	return nil
}
func (m *mockRelayCommandSenderConfigurable) SendAutopilotControl(int64, *runnerv1.AutopilotControlCommand) error {
	return nil
}

// mockGeoResolver implements geo.Resolver for testing.
type mockGeoResolver struct {
	resolveFn func(ip string) *geo.Location
}

func (m *mockGeoResolver) Resolve(ip string) *geo.Location {
	if m.resolveFn != nil {
		return m.resolveFn(ip)
	}
	return nil
}

func (m *mockGeoResolver) Close() error {
	return nil
}
