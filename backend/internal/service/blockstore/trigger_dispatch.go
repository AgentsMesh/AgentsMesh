package blockstoreservice

import (
	"context"
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

func (s *Service) dispatchTriggers(ctx context.Context, wsID uuid.UUID, ops []*blockstore.BlockOp) {
	triggers, err := s.loadTriggers(ctx, wsID)
	if err != nil {
		s.logger.Warn("blockstore.trigger.load_failed", "err", err.Error())
		return
	}
	if len(triggers) == 0 {
		return
	}
	for _, op := range ops {
		if op.TargetBlock == nil {
			continue
		}
		event := opToTriggerEvent(op.Op)
		if event == "" {
			continue
		}
		target, err := s.repo.GetBlock(ctx, *op.TargetBlock)
		if err != nil {
			continue
		}
		for _, t := range triggers {
			if !t.Enabled || t.On != event || t.TargetType != target.Type {
				continue
			}
			if t.Predicate != "" && !evalTriggerPredicate(t.Predicate, target.Data) {
				continue
			}
			go s.fireTrigger(context.Background(), t, target, op)
		}
	}
}

func (s *Service) fireTrigger(ctx context.Context, t TriggerDef, target *blockstore.Block, op *blockstore.BlockOp) {
	switch t.Action.Kind {
	case "webhook":
		s.fireWebhook(ctx, t, target, op)
	case "agent":
		s.fireAgentAction(ctx, t, target, op)
	default:
		s.logger.Warn("blockstore.trigger.unknown_action_kind",
			"kind", t.Action.Kind, "trigger", t.Name)
	}
}

func opToTriggerEvent(op string) string {
	switch op {
	case blockstore.OpCreateBlock:
		return "create"
	case blockstore.OpUpdateBlock:
		return "update"
	case blockstore.OpDeleteBlock:
		return "delete"
	default:
		return ""
	}
}

func (s *Service) loadTriggers(ctx context.Context, wsID uuid.UUID) ([]TriggerDef, error) {
	def := blockstore.BlockTypeTriggerDef
	blocks, _, err := s.repo.ListBlocks(ctx, blockstore.BlockFilter{
		WorkspaceID: wsID,
		Type:        &def,
	})
	if err != nil {
		return nil, err
	}
	out := make([]TriggerDef, 0, len(blocks))
	for _, b := range blocks {
		t, ok := decodeTrigger(b.Data)
		if !ok {
			continue
		}
		t.CreatedBy = b.CreatedBy
		out = append(out, t)
	}
	return out, nil
}

func decodeTrigger(data blockstore.JSONMap) (TriggerDef, bool) {
	raw, err := json.Marshal(data)
	if err != nil {
		return TriggerDef{}, false
	}
	var t TriggerDef
	if err := json.Unmarshal(raw, &t); err != nil {
		return TriggerDef{}, false
	}
	if t.Name == "" || t.TargetType == "" || t.On == "" || t.Action.Kind == "" {
		return TriggerDef{}, false
	}
	return t, true
}
