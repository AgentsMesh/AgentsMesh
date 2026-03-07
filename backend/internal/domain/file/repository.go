package file

import "context"

// FileRepository defines the data access interface for file operations.
type FileRepository interface {
	// Create inserts a new file record.
	Create(ctx context.Context, f *File) error

	// GetByID returns a file by ID scoped to an organization.
	// Returns (nil, nil) if not found.
	GetByID(ctx context.Context, id, orgID int64) (*File, error)

	// Delete removes a file record.
	Delete(ctx context.Context, f *File) error
}
