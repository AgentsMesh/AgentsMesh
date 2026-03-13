// Package runnerlog provides the service layer for runner diagnostic log uploads.
package runnerlog

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runnerlog"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/google/uuid"
)

// presignPutExpiry is the default expiry for presigned PUT URLs (30 minutes).
const presignPutExpiry = 30 * time.Minute

// downloadURLExpiry is the default expiry for download URLs (1 hour).
const downloadURLExpiry = 1 * time.Hour

// Service handles runner log upload operations.
type Service struct {
	repo    runnerlog.Repository
	storage storage.Storage
}

// NewService creates a new runner log upload service.
func NewService(repo runnerlog.Repository, s storage.Storage) *Service {
	return &Service{repo: repo, storage: s}
}

// UploadRequest contains the result of a log upload request initiation.
type UploadRequest struct {
	RequestID    string `json:"request_id"`
	PresignedURL string `json:"presigned_url"`
	ExpiresAt    int64  `json:"expires_at"` // Unix seconds
}

// RequestUpload initiates a log upload request: creates DB record and presigned PUT URL.
func (s *Service) RequestUpload(ctx context.Context, orgID, runnerID, userID int64) (*UploadRequest, error) {
	requestID := uuid.New().String()
	now := time.Now()
	storageKey := fmt.Sprintf("orgs/%d/runner-logs/%d/%d/%s.tar.gz",
		orgID, now.Year(), int(now.Month()), fmt.Sprintf("%d_%s", runnerID, requestID))

	// Create DB record first (before generating presigned URL) so that
	// a failed URL generation doesn't leave orphaned S3 objects.
	record := &runnerlog.RunnerLog{
		OrganizationID: orgID,
		RunnerID:       runnerID,
		RequestID:      requestID,
		StorageKey:     storageKey,
		Status:         runnerlog.StatusPending,
		RequestedByID:  userID,
		CreatedAt:      now,
	}
	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create log record: %w", err)
	}

	// Generate presigned PUT URL
	// Use public presigned URL because Runner is self-hosted and connects from external networks
	presignedURL, err := s.storage.PresignPutURL(ctx, storageKey, "application/gzip", presignPutExpiry)
	if err != nil {
		// Mark the record as failed since we can't proceed
		_ = s.repo.MarkFailed(ctx, requestID, "failed to generate upload URL")
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := now.Add(presignPutExpiry).Unix()
	return &UploadRequest{
		RequestID:    requestID,
		PresignedURL: presignedURL,
		ExpiresAt:    expiresAt,
	}, nil
}

// HandleUploadStatus updates the log record based on Runner status reports.
// runnerID is used to verify the reporting runner owns the record.
func (s *Service) HandleUploadStatus(runnerID int64, requestID, phase string, progress int32, message, errMsg string, sizeBytes int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate phase
	if !runnerlog.ValidStatuses[phase] {
		slog.Warn("Unknown log upload phase", "request_id", requestID, "phase", phase)
		return
	}

	if err := s.repo.UpdateStatus(ctx, requestID, runnerID, phase, sizeBytes, errMsg); err != nil {
		slog.Error("Failed to update log upload status",
			"request_id", requestID,
			"runner_id", runnerID,
			"phase", phase,
			"error", err,
		)
	}
}

// MarkFailed sets a log record to failed status. Used when gRPC command send fails.
func (s *Service) MarkFailed(requestID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.repo.MarkFailed(ctx, requestID, reason); err != nil {
		slog.Error("Failed to mark log upload as failed",
			"request_id", requestID,
			"error", err,
		)
	}
}

// LogEntry represents a log record with optional download URL for API response.
type LogEntry struct {
	*runnerlog.RunnerLog
	DownloadURL string `json:"download_url,omitempty"`
}

// ListByRunner returns log records for a runner with download URLs for completed entries.
func (s *Service) ListByRunner(ctx context.Context, orgID, runnerID int64, limit, offset int) ([]*LogEntry, error) {
	logs, err := s.repo.ListByRunner(ctx, orgID, runnerID, limit, offset)
	if err != nil {
		return nil, err
	}

	entries := make([]*LogEntry, len(logs))
	for i, l := range logs {
		entry := &LogEntry{RunnerLog: l}
		if l.Status == runnerlog.StatusCompleted && l.StorageKey != "" {
			downloadURL, err := s.storage.GetURL(ctx, l.StorageKey, downloadURLExpiry)
			if err != nil {
				slog.Warn("Failed to generate download URL",
					"request_id", l.RequestID,
					"error", err,
				)
			} else {
				entry.DownloadURL = downloadURL
			}
		}
		entries[i] = entry
	}
	return entries, nil
}
