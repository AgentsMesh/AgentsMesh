package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// logUploadInProgress guards against concurrent upload operations.
var logUploadInProgress atomic.Bool

// uploadHTTPClient is a dedicated client for log uploads with proper timeouts.
var uploadHTTPClient = &http.Client{
	Timeout: 5 * time.Minute,
}

// OnUploadLogs handles log upload command from server.
func (h *RunnerMessageHandler) OnUploadLogs(cmd *runnerv1.UploadLogsCommand) error {
	log := logger.Runner()
	log.Info("Received upload logs command", "request_id", cmd.RequestId)

	// H3: prevent concurrent uploads
	if !logUploadInProgress.CompareAndSwap(false, true) {
		log.Info("Log upload already in progress, discarding command", "request_id", cmd.RequestId)
		return nil
	}
	defer logUploadInProgress.Store(false)

	// H1: validate presigned URL scheme (only allow http/https)
	parsedURL, err := url.Parse(cmd.PresignedUrl)
	if err != nil {
		h.sendLogUploadStatus(cmd.RequestId, "failed", 0, "", "invalid upload URL", 0)
		return fmt.Errorf("invalid presigned URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		h.sendLogUploadStatus(cmd.RequestId, "failed", 0, "", "invalid upload URL", 0)
		return fmt.Errorf("invalid presigned URL scheme: %s", parsedURL.Scheme)
	}

	// Check if runner is shutting down
	runCtx := h.runner.GetRunContext()
	if runCtx.Err() != nil {
		h.sendLogUploadStatus(cmd.RequestId, "failed", 0, "", "runner is shutting down", 0)
		return fmt.Errorf("runner is shutting down")
	}

	// H2: use url_expires_at to bound the context deadline
	timeout := 10 * time.Minute
	if cmd.UrlExpiresAt > 0 {
		remaining := time.Until(time.Unix(cmd.UrlExpiresAt, 0))
		if remaining <= 0 {
			h.sendLogUploadStatus(cmd.RequestId, "failed", 0, "", "presigned URL already expired", 0)
			return fmt.Errorf("presigned URL already expired")
		}
		if remaining < timeout {
			timeout = remaining
		}
	}

	ctx, cancel := context.WithTimeout(runCtx, timeout)
	defer cancel()

	// Phase: collecting
	h.sendLogUploadStatus(cmd.RequestId, "collecting", 0, "Collecting log files...", "", 0)

	tarPath, sizeBytes, err := CollectLogs(ctx)
	if err != nil {
		h.sendLogUploadStatus(cmd.RequestId, "failed", 0, "", fmt.Sprintf("failed to collect logs: %v", err), 0)
		return fmt.Errorf("failed to collect logs: %w", err)
	}
	defer os.Remove(tarPath)

	// Phase: uploading
	h.sendLogUploadStatus(cmd.RequestId, "uploading", 50, "Uploading log archive...", "", sizeBytes)

	if err := uploadLogArchive(ctx, cmd.PresignedUrl, tarPath); err != nil {
		h.sendLogUploadStatus(cmd.RequestId, "failed", 0, "", fmt.Sprintf("failed to upload logs: %v", err), sizeBytes)
		return fmt.Errorf("failed to upload logs: %w", err)
	}

	// Phase: completed
	h.sendLogUploadStatus(cmd.RequestId, "completed", 100, "Log upload completed", "", sizeBytes)
	log.Info("Log upload completed", "request_id", cmd.RequestId, "size_bytes", sizeBytes)
	return nil
}

// uploadLogArchive uploads the tar.gz archive to the presigned URL via HTTP PUT.
func uploadLogArchive(ctx context.Context, presignedURL, tarPath string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat archive: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, f)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.ContentLength = info.Size()
	req.Header.Set("Content-Type", "application/gzip")

	// H4: use dedicated client with timeout instead of http.DefaultClient
	resp, err := uploadHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	// M4: fully consume the response body so the HTTP connection can be reused
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload returned status %d", resp.StatusCode)
	}

	return nil
}

// sendLogUploadStatus sends a log upload status event to the server.
func (h *RunnerMessageHandler) sendLogUploadStatus(requestID, phase string, progress int32, message, errMsg string, sizeBytes int64) {
	event := &runnerv1.LogUploadStatusEvent{
		RequestId: requestID,
		Phase:     phase,
		Progress:  progress,
		Message:   message,
		Error:     errMsg,
		SizeBytes: sizeBytes,
	}

	if err := h.conn.SendLogUploadStatus(event); err != nil {
		logger.Runner().Error("Failed to send log upload status",
			"request_id", requestID,
			"phase", phase,
			"error", err,
		)
	}
}
