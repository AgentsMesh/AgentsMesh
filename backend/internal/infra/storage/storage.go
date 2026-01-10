// Package storage provides object storage abstraction for S3-compatible services.
package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a stored file.
type FileInfo struct {
	Key         string    // Storage path/key
	Size        int64     // File size in bytes
	ContentType string    // MIME type
	ETag        string    // File hash/etag
	LastModified time.Time // Last modification time
}

// Storage defines the interface for object storage operations.
// Implementations should support S3-compatible services (AWS S3, MinIO, OSS in S3 mode).
type Storage interface {
	// Upload stores a file and returns its metadata.
	// key: storage path (e.g., "orgs/1/files/2024/01/abc123.png")
	// reader: file content stream
	// size: file size in bytes (use -1 if unknown)
	// contentType: MIME type (e.g., "image/png")
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*FileInfo, error)

	// Delete removes a file from storage.
	Delete(ctx context.Context, key string) error

	// GetURL returns a URL for accessing the file.
	// For private buckets, returns a pre-signed URL valid for the specified duration.
	// For public buckets, returns a direct URL (expiry is ignored).
	GetURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// Exists checks if a file exists in storage.
	Exists(ctx context.Context, key string) (bool, error)
}
