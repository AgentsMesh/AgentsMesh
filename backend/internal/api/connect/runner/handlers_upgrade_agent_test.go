package runnerconnect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rundom "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	runner "github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	runnerapiv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner_api/v1"
)

type fakeUpgradeSender struct {
	connected    bool
	sendAgentErr error
	sentSlug     string
	sentCalled   bool
}

func (f *fakeUpgradeSender) IsConnected(int64) bool                              { return f.connected }
func (f *fakeUpgradeSender) SendUpgradeRunner(int64, string, string, bool) error { return nil }
func (f *fakeUpgradeSender) SendUpgradeAgent(_ int64, _, slug, _ string, _ []string) error {
	f.sentCalled = true
	f.sentSlug = slug
	return f.sendAgentErr
}

type fakeAgentUpgrade struct {
	exe  string
	argv []string
	ok   bool
	err  error
}

func (f *fakeAgentUpgrade) GetUpgradeCommand(context.Context, string) (string, []string, bool, error) {
	return f.exe, f.argv, f.ok, f.err
}

// newUpgradeAgentServer wires a Server over a test DB with one online runner in
// org 7 (matching fakeOrgService's org id) and an admin tenant (full write),
// plus the supplied sender + agent-command reader.
func newUpgradeAgentServer(t *testing.T, sender *fakeUpgradeSender, au *fakeAgentUpgrade) (*Server, int64) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	svc := runner.NewService(infra.NewRunnerRepository(db))
	r := &rundom.Runner{OrganizationID: 7, NodeID: "node-1", Status: rundom.RunnerStatusOnline}
	require.NoError(t, db.Create(r).Error)
	srv := NewServer(svc, &fakeOrgService{role: "admin"},
		WithUpgradeCommandSender(sender), WithAgentUpgradeReader(au))
	return srv, r.ID
}

// --- guards (return before GetRunner, no DB needed) ---

func TestUpgradeAgent_ServiceNotConfigured_Unavailable(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "admin"})
	_, err := srv.UpgradeAgent(ctxAsUser(42),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{OrgSlug: "acme", Id: 1, AgentSlug: "claude-code"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnavailable, connectCodeOf(t, err))
}

func TestUpgradeAgent_MissingOrgSlug_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "admin"},
		WithUpgradeCommandSender(&fakeUpgradeSender{connected: true}),
		WithAgentUpgradeReader(&fakeAgentUpgrade{ok: true, exe: "claude", argv: []string{"npm"}}))
	_, err := srv.UpgradeAgent(ctxAsUser(42),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{AgentSlug: "claude-code"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestUpgradeAgent_NoAuth_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, &fakeOrgService{role: "admin"},
		WithUpgradeCommandSender(&fakeUpgradeSender{connected: true}),
		WithAgentUpgradeReader(&fakeAgentUpgrade{ok: true}))
	_, err := srv.UpgradeAgent(context.Background(),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{OrgSlug: "acme", AgentSlug: "claude-code"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connectCodeOf(t, err))
}

// --- full path (DB-backed runner + admin tenant) ---

func TestUpgradeAgent_Success_SendsCommand(t *testing.T) {
	sender := &fakeUpgradeSender{connected: true}
	srv, id := newUpgradeAgentServer(t, sender, &fakeAgentUpgrade{ok: true, exe: "claude", argv: []string{"npm", "i"}})
	resp, err := srv.UpgradeAgent(ctxAsUser(42),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{OrgSlug: "acme", Id: id, AgentSlug: "claude-code"}))
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Msg.GetRequestId())
	assert.True(t, sender.sentCalled, "SendUpgradeAgent should be called on success")
	assert.Equal(t, "claude-code", sender.sentSlug)
}

func TestUpgradeAgent_Unsupported_InvalidArgument(t *testing.T) {
	sender := &fakeUpgradeSender{connected: true}
	srv, id := newUpgradeAgentServer(t, sender, &fakeAgentUpgrade{ok: false})
	_, err := srv.UpgradeAgent(ctxAsUser(42),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{OrgSlug: "acme", Id: id, AgentSlug: "cursor-cli"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
	assert.False(t, sender.sentCalled, "no command should be sent for an unsupported agent")
}

func TestUpgradeAgent_ReaderError_Internal(t *testing.T) {
	sender := &fakeUpgradeSender{connected: true}
	srv, id := newUpgradeAgentServer(t, sender, &fakeAgentUpgrade{err: errors.New("db down")})
	_, err := srv.UpgradeAgent(ctxAsUser(42),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{OrgSlug: "acme", Id: id, AgentSlug: "x"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connectCodeOf(t, err))
}

func TestUpgradeAgent_RunnerOffline_Unavailable(t *testing.T) {
	sender := &fakeUpgradeSender{connected: false}
	srv, id := newUpgradeAgentServer(t, sender, &fakeAgentUpgrade{ok: true, exe: "claude", argv: []string{"npm"}})
	_, err := srv.UpgradeAgent(ctxAsUser(42),
		connect.NewRequest(&runnerapiv1.UpgradeAgentRequest{OrgSlug: "acme", Id: id, AgentSlug: "x"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnavailable, connectCodeOf(t, err))
}
