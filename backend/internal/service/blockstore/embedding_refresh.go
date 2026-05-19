package blockstoreservice

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

func (s *Service) refreshEmbeddings(ctx context.Context, ops []*blockstore.BlockOp) {
	for _, op := range ops {
		switch op.Op {
		case blockstore.OpCreateBlock, blockstore.OpUpdateBlock:
			if op.TargetBlock == nil {
				continue
			}
			s.embedBlock(ctx, *op.TargetBlock)
		case blockstore.OpDeleteBlock:
			if op.TargetBlock == nil {
				continue
			}
			if err := s.repo.DeleteEmbedding(ctx, *op.TargetBlock); err != nil {
				s.logger.Warn("blockstore.embedding.delete_failed",
					"block_id", op.TargetBlock, "err", err.Error())
			}
		}
	}
}

func (s *Service) embedBlock(ctx context.Context, blockID uuid.UUID) {
	b, err := s.repo.GetBlock(ctx, blockID)
	if err != nil {
		s.logger.Warn("blockstore.embedding.block_fetch_failed",
			"block_id", blockID, "err", err.Error())
		return
	}
	text := ""
	if b.Text != nil {
		text = *b.Text
	}
	if text == "" {
		if err := s.repo.DeleteEmbedding(ctx, blockID); err != nil {
			s.logger.Warn("blockstore.embedding.clear_failed",
				"block_id", blockID, "err", err.Error())
		}
		return
	}
	newHash := HashTextForEmbedding(text)
	existingHash, err := s.repo.GetEmbeddingHash(ctx, blockID)
	if err == nil && existingHash == newHash {
		return
	}
	vec, err := s.embedder.Embed(ctx, text)
	if err != nil {
		s.logger.Warn("blockstore.embedding.embed_failed",
			"block_id", blockID, "err", err.Error())
		return
	}
	if err := s.repo.UpsertEmbedding(
		ctx, blockID, s.embedder.Model(), s.embedder.Dims(), vec, newHash,
	); err != nil {
		s.logger.Warn("blockstore.embedding.upsert_failed",
			"block_id", blockID, "err", err.Error())
	}
}
