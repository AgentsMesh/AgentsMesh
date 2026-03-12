package runnerlog

import "context"

// Repository defines the data access interface for runner logs.
type Repository interface {
	Create(ctx context.Context, log *RunnerLog) error
	GetByRequestID(ctx context.Context, requestID string) (*RunnerLog, error)

	// UpdateStatus atomically transitions the record status.
	// Only non-zero sizeBytes and non-empty errorMessage are written.
	// Terminal states (completed/failed) cannot be overwritten.
	// runnerID is validated against the record to prevent cross-runner spoofing.
	// Returns ErrNotFound if no matching record exists, ErrStaleStatus if already terminal.
	UpdateStatus(ctx context.Context, requestID string, runnerID int64, status string, sizeBytes int64, errorMessage string) error

	// MarkFailed sets a record to failed status. Used when gRPC send fails.
	MarkFailed(ctx context.Context, requestID string, errorMessage string) error

	ListByRunner(ctx context.Context, orgID, runnerID int64, limit, offset int) ([]*RunnerLog, error)
}
