package binding

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupIntegrationService creates a binding Service with an in-memory DB and
// nil PodQuerier (manual approval mode). Returns service, db, and context.
func setupIntegrationService(t *testing.T) (*Service, *gorm.DB, context.Context) {
	t.Helper()
	db := setupTestDB(t)
	svc := newTestService(db, nil)
	return svc, db, context.Background()
}

func TestBinding_RequestAndAccept(t *testing.T) {
	svc, _, ctx := setupIntegrationService(t)

	// Request a binding — nil podQuerier means pending
	binding, err := svc.RequestBinding(ctx, 1, "podA", "podB",
		[]string{channel.BindingScopePodRead}, "")
	require.NoError(t, err)
	assert.Equal(t, channel.BindingStatusPending, binding.Status)
	assert.Contains(t, []string(binding.PendingScopes), channel.BindingScopePodRead)

	// Accept the binding
	accepted, err := svc.AcceptBinding(ctx, binding.ID, "podB")
	require.NoError(t, err)
	assert.Equal(t, channel.BindingStatusActive, accepted.Status)
	assert.Contains(t, []string(accepted.GrantedScopes), channel.BindingScopePodRead)
	assert.Empty(t, accepted.PendingScopes)
}

func TestBinding_RequestAndReject(t *testing.T) {
	svc, _, ctx := setupIntegrationService(t)

	binding, err := svc.RequestBinding(ctx, 1, "rejA", "rejB",
		[]string{channel.BindingScopePodRead}, channel.BindingPolicyExplicitOnly)
	require.NoError(t, err)
	assert.Equal(t, channel.BindingStatusPending, binding.Status)

	rejected, err := svc.RejectBinding(ctx, binding.ID, "rejB", "not needed")
	require.NoError(t, err)
	assert.Equal(t, channel.BindingStatusRejected, rejected.Status)
	require.NotNil(t, rejected.RejectionReason)
	assert.Equal(t, "not needed", *rejected.RejectionReason)
}

func TestBinding_AutoBinding(t *testing.T) {
	svc, _, ctx := setupIntegrationService(t)

	binding, err := svc.CreateAutoBinding(ctx, 1, "autoA", "autoB",
		[]string{channel.BindingScopePodRead, channel.BindingScopePodWrite})
	require.NoError(t, err)
	assert.Equal(t, channel.BindingStatusActive, binding.Status)
	assert.Len(t, binding.GrantedScopes, 2)
	assert.Contains(t, []string(binding.GrantedScopes), channel.BindingScopePodRead)
	assert.Contains(t, []string(binding.GrantedScopes), channel.BindingScopePodWrite)
}

func TestBinding_Unbind(t *testing.T) {
	svc, _, ctx := setupIntegrationService(t)

	_, err := svc.CreateAutoBinding(ctx, 1, "unbA", "unbB",
		[]string{channel.BindingScopePodRead})
	require.NoError(t, err)

	success, err := svc.Unbind(ctx, "unbA", "unbB")
	require.NoError(t, err)
	assert.True(t, success)

	bound, err := svc.IsBound(ctx, "unbA", "unbB")
	require.NoError(t, err)
	assert.False(t, bound)
}

func TestBinding_ScopeCheck(t *testing.T) {
	svc, _, ctx := setupIntegrationService(t)

	_, err := svc.CreateAutoBinding(ctx, 1, "scA", "scB",
		[]string{channel.BindingScopePodRead, channel.BindingScopePodWrite})
	require.NoError(t, err)

	hasRead, err := svc.HasScope(ctx, "scA", "scB", channel.BindingScopePodRead)
	require.NoError(t, err)
	assert.True(t, hasRead)

	// "admin" is not a valid binding scope, so it should return false
	hasAdmin, err := svc.HasScope(ctx, "scA", "scB", "admin")
	require.NoError(t, err)
	assert.False(t, hasAdmin)
}

func TestBinding_SelfBindingError(t *testing.T) {
	svc, _, ctx := setupIntegrationService(t)

	_, err := svc.RequestBinding(ctx, 1, "selfPod", "selfPod",
		[]string{channel.BindingScopePodRead}, "")
	assert.ErrorIs(t, err, ErrSelfBinding)
}

func TestBinding_CleanupExpired(t *testing.T) {
	svc, db, ctx := setupIntegrationService(t)

	// Create a pending binding
	_, err := svc.RequestBinding(ctx, 1, "expA", "expB",
		[]string{channel.BindingScopePodRead}, channel.BindingPolicyExplicitOnly)
	require.NoError(t, err)

	// Manually set expires_at to the past
	result := db.Exec(
		"UPDATE pod_bindings SET expires_at = datetime('now', '-1 day') WHERE initiator_pod = ?",
		"expA",
	)
	require.NoError(t, result.Error)

	count, err := svc.CleanupExpiredBindings(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// The binding should now be expired, not pending
	binding, err := svc.GetBinding(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, channel.BindingStatusExpired, binding.Status)
}
