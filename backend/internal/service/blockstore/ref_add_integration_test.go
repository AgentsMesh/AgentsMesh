package blockstoreservice

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddRef_MetaRoundTrip guards against the regression where ref.meta was
// accepted by the service but stripped from op.forward, so collaborators
// receiving the op via WS would reconstruct the ref with empty meta.
//
// The fix: ref_add.go now writes meta into forward; this test asserts both
// the DB row and the op envelope carry the caller's meta verbatim.
func TestAddRef_MetaRoundTrip(t *testing.T) {
	svc, actor, wsID, rootID := setup(t)
	ctx := context.Background()

	child := uuid.New()
	_, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id": child.String(), "type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "x", "status": "todo"},
			}},
			{Op: blockstore.OpAddRef, Payload: map[string]any{
				"from": rootID.String(), "to": child.String(),
				"rel":       blockstore.RelNest,
				"order_key": "a0",
				"meta":      map[string]any{"source": "agent-42", "confidence": 0.8},
			}},
		},
	})
	require.NoError(t, err)

	// DB row carries the meta.
	refs, err := svc.repo.ListRefs(ctx, blockstore.RefFilter{
		WorkspaceID: wsID,
		FromID:      &rootID,
	})
	require.NoError(t, err)
	var addRefEntry *blockstore.BlockRef
	for _, r := range refs {
		if r.ToID == child {
			addRefEntry = r
			break
		}
	}
	require.NotNil(t, addRefEntry)
	assert.Equal(t, "agent-42", addRefEntry.Meta["source"])

	// Op forward carries the meta so WS subscribers reconstruct correctly.
	ops, err := svc.repo.StreamOps(ctx, blockstore.OpStreamFilter{
		WorkspaceID: wsID, Limit: 100,
	})
	require.NoError(t, err)
	var addOp *blockstore.BlockOp
	for _, op := range ops {
		if op.Op == blockstore.OpAddRef {
			addOp = op
		}
	}
	require.NotNil(t, addOp, "expected an addRef op in the stream")
	meta, ok := addOp.Forward["meta"].(map[string]any)
	require.True(t, ok, "forward.meta missing or wrong type: %+v", addOp.Forward)
	assert.Equal(t, "agent-42", meta["source"])
}

// TestUpdateRef_BumpsUpdatedAt guards the BlockRef.UpdatedAt field added in
// migration 000119. Moving a ref (change order_key) must advance updated_at
// so audit / backlink UIs can sort by last-touched.
func TestUpdateRef_BumpsUpdatedAt(t *testing.T) {
	svc, actor, wsID, rootID := setup(t)
	ctx := context.Background()

	child := uuid.New()
	_, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id": child.String(), "type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "x", "status": "todo"},
			}},
			{Op: blockstore.OpAddRef, Payload: map[string]any{
				"from": rootID.String(), "to": child.String(),
				"rel": blockstore.RelNest, "order_key": "a0",
			}},
		},
	})
	require.NoError(t, err)

	refs, err := svc.repo.ListRefs(ctx, blockstore.RefFilter{
		WorkspaceID: wsID, FromID: &rootID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, refs)
	createdAt := refs[0].UpdatedAt

	// Ensure at least one clock tick so updated_at can strictly advance.
	time.Sleep(1100 * time.Millisecond)

	_, err = svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpUpdateRef, Payload: map[string]any{
				"ref_id":    refs[0].ID,
				"order_key": "a1",
			}},
		},
	})
	require.NoError(t, err)

	refs2, err := svc.repo.ListRefs(ctx, blockstore.RefFilter{
		WorkspaceID: wsID, FromID: &rootID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, refs2)
	assert.True(t, refs2[0].UpdatedAt.After(createdAt),
		"updated_at should advance: before=%s after=%s",
		createdAt, refs2[0].UpdatedAt)
}

// TestUpdateRef_MetaRoundTrip guards the Phase 4+ capability to mutate
// ref.meta via updateRef — critical for workflow like "mark a comment ref
// as resolved" or "record who reordered this nest edge".
func TestUpdateRef_MetaRoundTrip(t *testing.T) {
	svc, actor, wsID, rootID := setup(t)
	ctx := context.Background()

	child := uuid.New()
	_, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id": child.String(), "type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "x", "status": "todo"},
			}},
			{Op: blockstore.OpAddRef, Payload: map[string]any{
				"from": rootID.String(), "to": child.String(),
				"rel": blockstore.RelNest, "order_key": "a0",
				"meta": map[string]any{"source": "initial"},
			}},
		},
	})
	require.NoError(t, err)

	refs, err := svc.repo.ListRefs(ctx, blockstore.RefFilter{
		WorkspaceID: wsID, FromID: &rootID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, refs)
	refID := refs[0].ID

	_, err = svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpUpdateRef, Payload: map[string]any{
				"ref_id": refID,
				"meta":   map[string]any{"source": "updated", "resolved": true},
			}},
		},
	})
	require.NoError(t, err)

	// DB row reflects the merged meta.
	refs, err = svc.repo.ListRefs(ctx, blockstore.RefFilter{
		WorkspaceID: wsID, FromID: &rootID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, refs)
	assert.Equal(t, "updated", refs[0].Meta["source"])
	assert.Equal(t, true, refs[0].Meta["resolved"])

	// The forward diff on the updateRef op carries the new meta so WS
	// subscribers rebuild the same state.
	ops, err := svc.repo.StreamOps(ctx, blockstore.OpStreamFilter{
		WorkspaceID: wsID, Limit: 100,
	})
	require.NoError(t, err)
	var updateOp *blockstore.BlockOp
	for _, op := range ops {
		if op.Op == blockstore.OpUpdateRef {
			updateOp = op
		}
	}
	require.NotNil(t, updateOp)
	meta, ok := updateOp.Forward["meta"].(blockstore.JSONMap)
	if !ok {
		// Some driver paths deserialize nested JSON as map[string]any.
		raw, okMap := updateOp.Forward["meta"].(map[string]any)
		require.True(t, okMap, "forward.meta type=%T value=%v", updateOp.Forward["meta"], updateOp.Forward["meta"])
		meta = blockstore.JSONMap(raw)
	}
	assert.Equal(t, "updated", meta["source"])
	assert.Equal(t, true, meta["resolved"])
}
