package blockstoreservice

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/infra/otel"
	"github.com/google/uuid"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/attribute"
)

// ApplyOps is the sole write entry point — entire batch runs in a
// workspace-scoped advisory-locked transaction; IdempotencyKey replay returns
// recorded op ids without re-execution.
func (s *Service) ApplyOps(ctx context.Context, actor ActorContext, in ApplyOpsInput) (*ApplyOpsResult, error) {
	s.WarnIfDefaultEmbedder()
	start := time.Now()
	defer func() {
		otel.BlockstoreOpsDuration.Record(ctx, float64(time.Since(start).Milliseconds()),
			otelmetric.WithAttributes(attribute.String("actor_type", actor.ActorType)))
	}()
	if len(in.Ops) == 0 {
		return nil, blockstore.ErrApplyOpsEmpty
	}
	wsID, err := uuid.Parse(in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	ws, err := s.repo.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	if ws.OrganizationID != actor.OrgID {
		return nil, blockstore.ErrOrgMismatch
	}

	var applied []*blockstore.BlockOp
	result := &ApplyOpsResult{ParentOpID: in.ParentOpID}

	err = s.repo.WithinWorkspaceTx(ctx, wsID, func(tx blockstore.TxWriter) error {
		if in.IdempotencyKey != "" {
			existing, err := tx.FindOpByIdempotencyKey(ctx, in.IdempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				result.WasReplay = true
				result.OpIDs = append(result.OpIDs, existing.ID)
				// Replay must return every op id from the original batch, not just the first.
				siblings, err := tx.ListOpsByParent(ctx, existing.ID)
				if err != nil {
					return err
				}
				for _, sib := range siblings {
					result.OpIDs = append(result.OpIDs, sib.ID)
				}
				return nil
			}
		}

		var parentOp *int64
		if in.ParentOpID != nil {
			parentOp = in.ParentOpID
		}
		var firstOpID int64

		for idx, envelope := range in.Ops {
			if err := ensureValidOp(envelope.Op); err != nil {
				return err
			}
			op, err := s.applyOne(ctx, tx, actor, envelope, wsID)
			if err != nil {
				return err
			}
			// IdempotencyKey attaches to op[0]; op[1..] link via parent_op_id so replay can reassemble.
			if idx == 0 {
				if in.IdempotencyKey != "" {
					op.IdempotencyKey = &in.IdempotencyKey
				}
				op.ParentOpID = parentOp
			} else {
				op.ParentOpID = &firstOpID
			}
			id, err := tx.InsertOp(ctx, op)
			if err != nil {
				return err
			}
			op.ID = id
			if idx == 0 {
				firstOpID = id
			}
			applied = append(applied, op)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, op := range applied {
		result.OpIDs = append(result.OpIDs, op.ID)
		otel.BlockstoreOpsApplied.Add(ctx, 1,
			otelmetric.WithAttributes(
				attribute.String("op_kind", op.Op),
				attribute.String("actor_type", actor.ActorType),
			))
	}

	if s.publisher != nil && !result.WasReplay {
		s.publisher.PublishBatch(ctx, actor.OrgID, applied)
	}
	if !result.WasReplay {
		s.enqueueEmbeddings(applied)
	}
	if !result.WasReplay && !in.SuppressTriggers {
		s.dispatchTriggers(ctx, wsID, applied)
	}
	return result, nil
}

func (s *Service) applyOne(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	env OpEnvelope,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	switch env.Op {
	case blockstore.OpCreateBlock:
		return s.applyCreateBlock(ctx, tx, actor, env.Payload, wsID)
	case blockstore.OpUpdateBlock:
		return s.applyUpdateBlock(ctx, tx, actor, env.Payload, wsID)
	case blockstore.OpDeleteBlock:
		return s.applyDeleteBlock(ctx, tx, actor, env.Payload, wsID)
	case blockstore.OpAddRef:
		return s.applyAddRef(ctx, tx, actor, env.Payload, wsID)
	case blockstore.OpRemoveRef:
		return s.applyRemoveRef(ctx, tx, actor, env.Payload, wsID)
	case blockstore.OpUpdateRef:
		return s.applyUpdateRef(ctx, tx, actor, env.Payload, wsID)
	default:
		return nil, blockstore.ErrUnknownOpKind
	}
}
