package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failure after the claim must roll it back, so the token's used_at returns to
// NULL and it stays reusable — the property that replaced the old defer-unclaim.
func TestReactivateWithTokenAtomic_RollsBackClaimOnCertFailure(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewRunnerRepository(db)
	ctx := context.Background()

	rn := &runner.Runner{OrganizationID: 1, NodeID: "react-node", Status: runner.RunnerStatusOffline}
	require.NoError(t, db.Create(rn).Error)

	tok := &runner.ReactivationToken{
		RunnerID: rn.ID, TokenHash: "h1", ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, db.Create(tok).Error)

	boom := errors.New("cert issuance failed")
	err := repo.ReactivateWithTokenAtomic(ctx, tok.ID, rn.ID, &runner.Certificate{}, func() error {
		return boom
	})
	require.ErrorIs(t, err, boom)

	var after runner.ReactivationToken
	require.NoError(t, db.First(&after, tok.ID).Error)
	assert.Nil(t, after.UsedAt, "the claim must roll back so the token stays reusable")

	var certs int64
	db.Model(&runner.Certificate{}).Where("runner_id = ?", rn.ID).Count(&certs)
	assert.Zero(t, certs, "no certificate may survive the rolled-back tx")
}

func TestReactivateWithTokenAtomic_RejectsAlreadyUsed(t *testing.T) {
	db := testkit.SetupTestDB(t)
	repo := NewRunnerRepository(db)
	ctx := context.Background()

	rn := &runner.Runner{OrganizationID: 1, NodeID: "react-node-2", Status: runner.RunnerStatusOffline}
	require.NoError(t, db.Create(rn).Error)

	used := time.Now()
	tok := &runner.ReactivationToken{
		RunnerID: rn.ID, TokenHash: "h2", ExpiresAt: time.Now().Add(10 * time.Minute), UsedAt: &used,
	}
	require.NoError(t, db.Create(tok).Error)

	err := repo.ReactivateWithTokenAtomic(ctx, tok.ID, rn.ID, &runner.Certificate{}, func() error {
		t.Fatal("issueCert must not run when the claim finds nothing")
		return nil
	})
	require.ErrorIs(t, err, runner.ErrReactivationUnavailable)
}
