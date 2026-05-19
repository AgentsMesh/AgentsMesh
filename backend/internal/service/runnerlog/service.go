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

const presignPutExpiry = 30 * time.Minute

const downloadURLExpiry = 1 * time.Hour

type Service struct {
	repo    runnerlog.Repository
	storage storage.Storage
}

func NewService(repo runnerlog.Repository, s storage.Storage) *Service {
	return &Service{repo: repo, storage: s}
}

type UploadRequest struct {
	RequestID    string `json:"request_id"`
	PresignedURL string `json:"presigned_url"`
	ExpiresAt    int64  `json:"expires_at"`
}

func (s *Service) RequestUpload(ctx context.Context, orgID, runnerID, userID int64) (*UploadRequest, error) {
	requestID := uuid.New().String()
	now := time.Now()
	storageKey := fmt.Sprintf("orgs/%d/runner-logs/%d/%d/%s.tar.gz",
		orgID, now.Year(), int(now.Month()), fmt.Sprintf("%d_%s", runnerID, requestID))

	// DB record before presigned URL — failed URL gen must not leave orphan S3 objects.
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

	// Public presigned URL — Runners are self-hosted and connect from external networks.
	presignedURL, err := s.storage.PresignPutURL(ctx, storageKey, "application/gzip", presignPutExpiry)
	if err != nil {
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

func (s *Service) HandleUploadStatus(runnerID int64, requestID, phase string, progress int32, message, errMsg string, sizeBytes int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !runnerlog.ValidStatuses[phase] {
		slog.WarnContext(ctx, "Unknown log upload phase", "request_id", requestID, "phase", phase)
		return
	}

	if err := s.repo.UpdateStatus(ctx, requestID, runnerID, phase, sizeBytes, errMsg); err != nil {
		slog.ErrorContext(ctx, "Failed to update log upload status",
			"request_id", requestID,
			"runner_id", runnerID,
			"phase", phase,
			"error", err,
		)
	}
}

func (s *Service) MarkFailed(requestID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.repo.MarkFailed(ctx, requestID, reason); err != nil {
		slog.ErrorContext(ctx, "Failed to mark log upload as failed",
			"request_id", requestID,
			"error", err,
		)
	}
}

type LogEntry struct {
	*runnerlog.RunnerLog
	DownloadURL string `json:"download_url,omitempty"`
}

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
				slog.WarnContext(ctx, "Failed to generate download URL",
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
