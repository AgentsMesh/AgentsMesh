package blockstoreservice

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

// End-to-end correlation: an ApplyOps with a trace id stamps that trace
// id onto every BlockOp it produces, and the trigger fire chain (webhook
// + agent_event) inherits the same id. This is the contract auditors
// rely on to stitch "user write → trigger → agent response" via a
// single trace_id, without joining op id chains.

// findOp pulls an op by id via StreamOps. Repository has no point-lookup;
// for tests we just stream the workspace and pick out the row we want.
func findOp(ctx context.Context, t *testing.T, svc *Service, wsID uuid.UUID, opID int64) *blockstore.BlockOp {
	t.Helper()
	ops, err := svc.repo.StreamOps(ctx, blockstore.OpStreamFilter{
		WorkspaceID: wsID,
		AfterID:     opID - 1,
		Limit:       1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, ops, "op id %d not found via StreamOps", opID)
	require.Equal(t, opID, ops[0].ID)
	return ops[0]
}

func TestApplyOps_StampsTraceIDOntoBlockOps(t *testing.T) {
	svc, baseActor, wsID, _ := setup(t)
	ctx := context.Background()

	actor := baseActor
	actor.TraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	actor.RequestID = actor.TraceID
	actor.IP = "10.0.0.5:51234"
	actor.UserAgent = "agentsmesh-cli/1.0"

	res, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id":   uuid.New().String(),
				"type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "trace-test", "status": "todo"},
			}},
		},
	})
	require.NoError(t, err)
	require.Len(t, res.OpIDs, 1)

	op := findOp(ctx, t, svc, wsID, res.OpIDs[0])
	require.NotNil(t, op.Context, "BlockOp.Context must be populated, not nil")
	assert.Equal(t, actor.TraceID, op.Context["trace_id"],
		"trace id from ActorContext must land in BlockOp.Context.trace_id")
	assert.Equal(t, actor.RequestID, op.Context["request_id"])
	assert.Equal(t, actor.IP, op.Context["ip"])
	assert.Equal(t, actor.UserAgent, op.Context["user_agent"])
}

func TestApplyOps_OmitsEmptyCorrelationFields(t *testing.T) {
	svc, actor, wsID, _ := setup(t)
	ctx := context.Background()
	// No correlation envelope on actor — service writes (no incoming HTTP/
	// gRPC request). Context map must omit absent keys so audit consumers
	// can rely on key presence as "we tried and observed it".

	res, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id":   uuid.New().String(),
				"type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "no-trace", "status": "todo"},
			}},
		},
	})
	require.NoError(t, err)

	op := findOp(ctx, t, svc, wsID, res.OpIDs[0])
	if op.Context != nil {
		_, hasTrace := op.Context["trace_id"]
		_, hasReq := op.Context["request_id"]
		assert.False(t, hasTrace, "no trace id on actor ⇒ no key in op context")
		assert.False(t, hasReq, "no request id on actor ⇒ no key in op context")
	}
}

func TestTrigger_WebhookCarriesTraceIDHeader(t *testing.T) {
	t.Setenv("BLOCKSTORE_WEBHOOK_ALLOW_HOSTS", "127.0.0.1")
	var fired atomic.Int32
	var gotTrace atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace.Store(r.Header.Get("X-Trace-Id"))
		_, _ = io.Copy(io.Discard, r.Body)
		fired.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc, actor, wsID, _ := setup(t)
	ctx := context.Background()

	_, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"type": blockstore.BlockTypeTriggerDef,
				"data": map[string]any{
					"name":        "trace-webhook",
					"target_type": blockstore.BlockTypeTask,
					"on":          "create",
					"enabled":     true,
					"action": map[string]any{
						"kind": "webhook",
						"url":  server.URL,
					},
				},
			}},
		},
	})
	require.NoError(t, err)

	traced := actor
	traced.TraceID = "deadbeefcafefacefeedfacecafebabe"
	traced.RequestID = traced.TraceID
	_, err = svc.ApplyOps(ctx, traced, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id":   uuid.New().String(),
				"type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "trace-trigger", "status": "todo"},
			}},
		},
	})
	require.NoError(t, err)

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && fired.Load() == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	require.Equal(t, int32(1), fired.Load(), "webhook should fire exactly once")
	header, _ := gotTrace.Load().(string)
	assert.Equal(t, "deadbeefcafefacefeedfacecafebabe", header,
		"webhook receiver must see X-Trace-Id matching the originating op")
}

func TestTrigger_AgentEventInheritsTraceID(t *testing.T) {
	svc, actor, wsID, _ := setup(t)
	ctx := context.Background()

	_, err := svc.ApplyOps(ctx, actor, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"type": blockstore.BlockTypeTriggerDef,
				"data": map[string]any{
					"name":        "trace-agent",
					"target_type": blockstore.BlockTypeTask,
					"on":          "create",
					"enabled":     true,
					"action": map[string]any{
						"kind":       "agent",
						"agent_slug": "trace-bot",
					},
				},
			}},
		},
	})
	require.NoError(t, err)

	traced := actor
	traced.TraceID = "1111111122222222333333334444aaaa"
	traced.RequestID = traced.TraceID
	_, err = svc.ApplyOps(ctx, traced, ApplyOpsInput{
		WorkspaceID: wsID.String(),
		Ops: []OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id":   uuid.New().String(),
				"type": blockstore.BlockTypeTask,
				"data": map[string]any{"title": "trace-agent-task", "status": "todo"},
			}},
		},
	})
	require.NoError(t, err)

	// Poll for the agent_event block — fireAgentAction is async.
	var events []*blockstore.Block
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		eventType := blockstore.BlockTypeAgentEvent
		found, _, ferr := svc.repo.ListBlocks(ctx, blockstore.BlockFilter{
			WorkspaceID: wsID,
			Type:        &eventType,
		})
		require.NoError(t, ferr)
		if len(found) > 0 {
			events = found
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	require.NotEmpty(t, events, "agent_event block should be written")

	// Walk every op produced in this workspace and locate the one that
	// targets the agent_event block. Its Context.trace_id must match the
	// originating user's trace id — that's the cross-actor inheritance
	// audit consumers rely on to stitch "user write → trigger → agent".
	ops, err := svc.repo.StreamOps(ctx, blockstore.OpStreamFilter{
		WorkspaceID: wsID,
		AfterID:     0,
		Limit:       1000,
	})
	require.NoError(t, err)
	var agentOp *blockstore.BlockOp
	for _, op := range ops {
		if op.TargetBlock != nil && *op.TargetBlock == events[0].ID {
			agentOp = op
			break
		}
	}
	require.NotNil(t, agentOp, "create op for agent_event block must exist")
	assert.Equal(t, traced.TraceID, agentOp.Context["trace_id"],
		"agent_event op must inherit the originating user's trace id")
}
