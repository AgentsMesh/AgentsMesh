package blockstoreservice

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type WorkspaceExport struct {
	Workspace *blockstore.BlockWorkspace `json:"workspace"`
	Blocks    []*blockstore.Block        `json:"blocks"`
	Refs      []*blockstore.BlockRef     `json:"refs"`
	Ops       []*blockstore.BlockOp      `json:"ops"`
	ExportedAt string                    `json:"exported_at"`
}

func (s *Service) ExportWorkspace(
	ctx context.Context,
	actor ActorContext,
	wsID uuid.UUID,
) (*WorkspaceExport, error) {
	ws, err := s.repo.GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, err
	}
	if ws.OrganizationID != actor.OrgID {
		return nil, blockstore.ErrOrgMismatch
	}

	blocks, _, err := s.repo.ListBlocks(ctx, blockstore.BlockFilter{
		WorkspaceID:    wsID,
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, err
	}

	refs, err := s.repo.ListRefs(ctx, blockstore.RefFilter{
		WorkspaceID: wsID,
	})
	if err != nil {
		return nil, err
	}

	var ops []*blockstore.BlockOp
	after := int64(0)
	for {
		page, err := s.repo.StreamOps(ctx, blockstore.OpStreamFilter{
			WorkspaceID: wsID, AfterID: after, Limit: 1000,
		})
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		ops = append(ops, page...)
		after = page[len(page)-1].ID
	}

	return &WorkspaceExport{
		Workspace:  ws,
		Blocks:     blocks,
		Refs:       refs,
		Ops:        ops,
		ExportedAt: nowISO(),
	}, nil
}

func nowISO() string { return currentTimeProvider() }

var currentTimeProvider = defaultTimeProvider

func defaultTimeProvider() string {
	return timeNowUTC().Format("2006-01-02T15:04:05Z07:00")
}
