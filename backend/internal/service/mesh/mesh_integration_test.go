package mesh

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMeshService creates a Service with nil sub-services (only repo-level ops).
func setupMeshService(t *testing.T) *Service {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	return NewService(repo, nil, nil, nil)
}

func TestMesh_JoinLeaveChannel(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	svc := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "mesh@test.io", "meshuser")
	orgID := testkit.CreateOrg(t, db, "mesh-org", userID)
	chID := testkit.CreateChannel(t, db, orgID, "general")
	runnerID := testkit.CreateRunner(t, db, orgID, "r-1")
	podKey := testkit.CreatePod(t, db, orgID, runnerID, userID)

	// Join
	err := svc.JoinChannel(ctx, chID, podKey)
	require.NoError(t, err)

	// Verify via repo helper
	keys, err := repo.GetChannelPodKeys(ctx, chID)
	require.NoError(t, err)
	assert.Contains(t, keys, podKey)

	// Leave
	err = svc.LeaveChannel(ctx, chID, podKey)
	require.NoError(t, err)

	keys, err = repo.GetChannelPodKeys(ctx, chID)
	require.NoError(t, err)
	assert.NotContains(t, keys, podKey)
}

func TestMesh_RecordChannelAccess(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	svc := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "access@test.io", "accessuser")
	orgID := testkit.CreateOrg(t, db, "access-org", userID)
	chID := testkit.CreateChannel(t, db, orgID, "logs")

	// Record access by user
	err := svc.RecordChannelAccess(ctx, chID, nil, &userID)
	require.NoError(t, err)

	// Verify record exists in DB
	var count int64
	db.Raw(`SELECT COUNT(*) FROM channel_access WHERE channel_id = ? AND user_id = ?`, chID, userID).Scan(&count)
	assert.Equal(t, int64(1), count)

	// Record access by pod
	runnerID := testkit.CreateRunner(t, db, orgID, "r-acc")
	podKey := testkit.CreatePod(t, db, orgID, runnerID, userID)
	err = svc.RecordChannelAccess(ctx, chID, &podKey, nil)
	require.NoError(t, err)

	db.Raw(`SELECT COUNT(*) FROM channel_access WHERE channel_id = ? AND pod_key = ?`, chID, podKey).Scan(&count)
	assert.Equal(t, int64(1), count)
}

func TestMesh_GetTopology_RequiresSubServices(t *testing.T) {
	svc := setupMeshService(t)
	ctx := context.Background()

	// GetTopology depends on podService which is nil — should panic or error.
	// This confirms that topology aggregation requires full wiring.
	assert.Panics(t, func() {
		_, _ = svc.GetTopology(ctx, 1)
	})
}

func TestMesh_JoinChannelDuplicate(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	svc := NewService(repo, nil, nil, nil)
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "dup@test.io", "dupuser")
	orgID := testkit.CreateOrg(t, db, "dup-org", userID)
	chID := testkit.CreateChannel(t, db, orgID, "dup-ch")
	runnerID := testkit.CreateRunner(t, db, orgID, "r-dup")
	podKey := testkit.CreatePod(t, db, orgID, runnerID, userID)

	require.NoError(t, svc.JoinChannel(ctx, chID, podKey))

	// Joining again should still succeed (SQLite doesn't enforce unique on this combo)
	err := svc.JoinChannel(ctx, chID, podKey)
	require.NoError(t, err)

	// Should have 2 rows (no unique constraint)
	keys, err := repo.GetChannelPodKeys(ctx, chID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestMesh_LeaveNonexistent(t *testing.T) {
	svc := setupMeshService(t)
	ctx := context.Background()

	// Leaving a non-member pod should not error (DELETE WHERE finds 0 rows)
	err := svc.LeaveChannel(ctx, 999, "pod-ghost")
	assert.NoError(t, err)
}

func TestMesh_ChannelPodKeys_Empty(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "empty@test.io", "emptyuser")
	orgID := testkit.CreateOrg(t, db, "empty-org", userID)
	chID := testkit.CreateChannel(t, db, orgID, "empty-ch")

	keys, err := repo.GetChannelPodKeys(ctx, chID)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestMesh_CountChannelMessages(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "msg@test.io", "msguser")
	orgID := testkit.CreateOrg(t, db, "msg-org", userID)
	chID := testkit.CreateChannel(t, db, orgID, "msg-ch")

	// Initially zero
	count, err := repo.CountChannelMessages(ctx, chID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Insert messages directly
	db.Exec(`INSERT INTO channel_messages (channel_id, message_type, content) VALUES (?, 'text', 'hello')`, chID)
	db.Exec(`INSERT INTO channel_messages (channel_id, message_type, content) VALUES (?, 'text', 'world')`, chID)

	count, err = repo.CountChannelMessages(ctx, chID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestMesh_ListPodsByTicketIDs(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := infra.NewMeshRepository(db)
	ctx := context.Background()

	userID := testkit.CreateUser(t, db, "ticket@test.io", "ticketuser")
	orgID := testkit.CreateOrg(t, db, "ticket-org", userID)
	runnerID := testkit.CreateRunner(t, db, orgID, "r-tkt")
	ticketID := testkit.CreateTicket(t, db, orgID, userID, "feat: foo")

	// Create a pod linked to the ticket
	podKey := testkit.CreatePod(t, db, orgID, runnerID, userID)
	db.Exec(`UPDATE pods SET ticket_id = ? WHERE pod_key = ?`, ticketID, podKey)

	pods, err := repo.ListPodsByTicketIDs(ctx, []int64{ticketID})
	require.NoError(t, err)
	require.Len(t, pods, 1)
	assert.Equal(t, podKey, pods[0].PodKey)
}
