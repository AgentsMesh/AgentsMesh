package runner

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRunnersVisibility(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	registrantUserID := int64(10)
	otherUserID := int64(20)

	orgRunner := &runner.Runner{
		OrganizationID:     1,
		NodeID:             "runner-org-vis",
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		IsEnabled:          true,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: &registrantUserID,
	}
	require.NoError(t, db.Create(orgRunner).Error)

	privateRunner := &runner.Runner{
		OrganizationID:     1,
		NodeID:             "runner-private-vis",
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		IsEnabled:          true,
		Visibility:         runner.VisibilityPrivate,
		RegisteredByUserID: &registrantUserID,
	}
	require.NoError(t, db.Create(privateRunner).Error)

	t.Run("registrant sees both org and private runners", func(t *testing.T) {
		runners, err := service.ListRunners(ctx, 1, registrantUserID)
		require.NoError(t, err)
		assert.Len(t, runners, 2)
	})

	t.Run("other user sees only org runner", func(t *testing.T) {
		runners, err := service.ListRunners(ctx, 1, otherUserID)
		require.NoError(t, err)
		assert.Len(t, runners, 1)
		assert.Equal(t, runner.VisibilityOrganization, runners[0].Visibility)
	})
}

func TestAuthorizeRunnerSetsRegisteredByUserID(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	org := createTestOrg(t, db, "test-org-regby")

	authKey := generateTestAuthKey()
	pendingAuth := &runner.PendingAuth{
		AuthKey:    authKey,
		MachineKey: "test-machine",
		ExpiresAt:  time.Now().Add(15 * time.Minute),
		Authorized: false,
	}
	require.NoError(t, db.Create(pendingAuth).Error)

	userID := int64(42)
	r, err := service.AuthorizeRunner(ctx, authKey, org.ID, userID, "my-registered-runner")
	require.NoError(t, err)
	assert.NotZero(t, r.ID)

	assert.NotNil(t, r.RegisteredByUserID)
	assert.Equal(t, userID, *r.RegisteredByUserID)
	assert.Equal(t, runner.VisibilityOrganization, r.Visibility)

	var dbRunner runner.Runner
	require.NoError(t, db.First(&dbRunner, r.ID).Error)
	assert.NotNil(t, dbRunner.RegisteredByUserID)
	assert.Equal(t, userID, *dbRunner.RegisteredByUserID)
	assert.Equal(t, runner.VisibilityOrganization, dbRunner.Visibility)
}

func TestSelectAvailableRunnerSkipsFullInCache(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	r1 := &runner.Runner{
		OrganizationID:    1,
		NodeID:            "runner-1",
		Description:       "Runner 1",
		Status:            runner.RunnerStatusOffline,
		MaxConcurrentPods: 2,
		IsEnabled:         true,
	}
	db.Create(r1)

	service.SetRunnerStatus(ctx, r1.ID, "online")
	db.Model(&runner.Runner{}).Where("id = ?", r1.ID).Update("current_pods", 2)
	r1Updated, _ := service.GetRunner(ctx, r1.ID)

	service.activeRunners.Store(r1.ID, &ActiveRunner{
		Runner:   r1Updated,
		LastPing: time.Now(),
		PodCount: 2,
	})

	_, err := service.SelectAvailableRunner(ctx, 1, 1)
	if err != ErrRunnerOffline {
		t.Errorf("expected ErrRunnerOffline for full runner, got %v", err)
	}
}
