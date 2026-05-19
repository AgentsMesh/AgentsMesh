package blockstore

import (
	"context"

	"github.com/google/uuid"
)

type RefFilter struct {
	WorkspaceID uuid.UUID
	FromID      *uuid.UUID
	ToID        *uuid.UUID
	Rel         *string
	Limit       int
	Offset      int
}

type BlockFilter struct {
	WorkspaceID uuid.UUID
	Type        *string
	IncludeDeleted bool
	Limit       int
	Offset      int
}

type OpStreamFilter struct {
	WorkspaceID uuid.UUID
	AfterID     int64 // exclusive
	Limit       int
}

type TxWriter interface {
	InsertBlock(ctx context.Context, b *Block) error
	UpdateBlockFields(ctx context.Context, id uuid.UUID, fields map[string]any) error
	SoftDeleteBlock(ctx context.Context, id uuid.UUID) error
	InsertRef(ctx context.Context, r *BlockRef) (int64, error)
	DeleteRefByID(ctx context.Context, id int64) error
	UpdateRefFields(ctx context.Context, id int64, fields map[string]any) error
	InsertOp(ctx context.Context, o *BlockOp) (int64, error)

	FindNestParent(ctx context.Context, childID uuid.UUID) (*BlockRef, error)
	FindAncestors(ctx context.Context, blockID uuid.UUID, maxDepth int) ([]uuid.UUID, error)
	FindRefByID(ctx context.Context, id int64) (*BlockRef, error)
	FindBlockByID(ctx context.Context, id uuid.UUID) (*Block, error)
	FindOpByIdempotencyKey(ctx context.Context, key string) (*BlockOp, error)
	// ListOpsByParent returns every op whose parent_op_id matches the given id,
	// ordered by id ascending. Used by idempotent replay to return the full
	// op list, not just the batch head. Empty result is a non-error.
	ListOpsByParent(ctx context.Context, parentOpID int64) ([]*BlockOp, error)
	ListTypeDefs(ctx context.Context) ([]*Block, error)
}

type Repository interface {
	CreateWorkspace(ctx context.Context, ws *BlockWorkspace) error
	GetWorkspace(ctx context.Context, id uuid.UUID) (*BlockWorkspace, error)
	GetWorkspaceBySlug(ctx context.Context, orgID int64, slug string) (*BlockWorkspace, error)
	ListWorkspaces(ctx context.Context, orgID int64) ([]*BlockWorkspace, error)
	UpdateWorkspaceRootBlock(ctx context.Context, wsID, rootID uuid.UUID) error
	DeleteWorkspaceCascade(ctx context.Context, workspaceID uuid.UUID) error

	WithinWorkspaceTx(ctx context.Context, workspaceID uuid.UUID, fn func(TxWriter) error) error

	GetBlock(ctx context.Context, id uuid.UUID) (*Block, error)
	ListBlocks(ctx context.Context, filter BlockFilter) ([]*Block, int64, error)
	ListChildren(ctx context.Context, parentID uuid.UUID, rel string) ([]*Block, []*BlockRef, error)
	ListBacklinks(ctx context.Context, targetID uuid.UUID, excludeNest bool) ([]*BlockRef, error)
	ListRefs(ctx context.Context, filter RefFilter) ([]*BlockRef, error)
	StreamOps(ctx context.Context, filter OpStreamFilter) ([]*BlockOp, error)

	ListWorkspaceSubtree(ctx context.Context, workspaceID, rootID uuid.UUID, maxDepth int) ([]*Block, []*BlockRef, error)

	GetTypeDefByKey(ctx context.Context, workspaceID uuid.UUID, typeKey string) (*Block, error)

	UpsertEmbedding(ctx context.Context, blockID uuid.UUID, model string, dims int, vector []float32, sourceHash string) error
	ListEmbeddings(ctx context.Context, workspaceID uuid.UUID, model string) ([]EmbeddingRow, error)
	DeleteEmbedding(ctx context.Context, blockID uuid.UUID) error
	GetEmbeddingHash(ctx context.Context, blockID uuid.UUID) (string, error)
	// SearchEmbeddings ranks and limits server-side when the Postgres vec
	// column + HNSW index are available; otherwise falls back to a projection
	// the service can sort in memory. Either way the caller gets at most topK
	// EmbeddingRows already narrowed to this workspace.
	SearchEmbeddings(ctx context.Context, workspaceID uuid.UUID, model string, queryVec []float32, topK int) ([]EmbeddingRow, error)
}

type EmbeddingRow struct {
	BlockID   uuid.UUID
	Type      string
	Text      *string
	Vector    []float32
	CreatedBy int64
	Meta      JSONMap
	Score float32
}
