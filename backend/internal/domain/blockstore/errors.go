package blockstore

import "errors"

var (
	ErrWorkspaceNotFound = errors.New("block workspace not found")
	ErrBlockNotFound     = errors.New("block not found")
	ErrRefNotFound       = errors.New("block ref not found")

	// ErrWorkspaceAlreadyExists is returned by CreateWorkspace when a UNIQUE
	// (org_id, slug) constraint violation prevents insertion. The service
	// layer catches this to turn a race during EnsureDefault into an
	// idempotent re-read instead of a 500.
	ErrWorkspaceAlreadyExists = errors.New("block workspace with this slug already exists")

	ErrUnknownBlockType   = errors.New("unknown block type")
	ErrUnknownOpKind      = errors.New("unknown op kind")
	ErrMissingRequiredKey = errors.New("missing required data key")
	ErrChildNotAllowed    = errors.New("child block type not allowed under parent")
	ErrInvalidRel         = errors.New("invalid ref rel")
	ErrOrderKeyRequired   = errors.New("order_key required for ordered rel")
	ErrCrossWorkspaceRef  = errors.New("ref to_id must live in same workspace as from_id")

	ErrStaleUpdate           = errors.New("block updated_at mismatch — stale write")
	ErrIdempotencyKeyReplay  = errors.New("idempotency key already processed")
	ErrSingleNestParent      = errors.New("block already has a nest parent")
	ErrNestCycle             = errors.New("nest ref would introduce a cycle")
	ErrApplyOpsEmpty         = errors.New("apply_ops payload must contain at least one op")
	ErrWorkspaceLockTimeout  = errors.New("workspace advisory lock acquisition timed out")

	ErrOrgMismatch    = errors.New("workspace does not belong to caller organization")
	ErrBlockForbidden = errors.New("actor is not authorized to access this block")

	ErrEmbeddingDisabled = errors.New("embedding provider not configured")

	ErrColumnValueInvalid = errors.New("column value failed validation")
)
