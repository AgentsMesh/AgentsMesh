package runnerlog

import (
	"testing"

	runnerlogDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runnerlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunnerLog_RequestAndUpdateStatus(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)
	require.NotEmpty(t, req.RequestID)
	require.NotEmpty(t, req.PresignedURL)

	// Transition to completed via HandleUploadStatus
	svc.HandleUploadStatus(runnerID, req.RequestID, runnerlogDomain.StatusCompleted, 100, "done", "", 2048)

	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, req.RequestID, entries[0].RequestID)
	assert.Equal(t, runnerlogDomain.StatusCompleted, entries[0].Status)
	assert.Equal(t, int64(2048), entries[0].SizeBytes)
	assert.NotNil(t, entries[0].CompletedAt)
	assert.NotEmpty(t, entries[0].DownloadURL)
}

func TestRunnerLog_MarkFailedIntegration(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)

	svc.MarkFailed(req.RequestID, "disk full")

	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, runnerlogDomain.StatusFailed, entries[0].Status)
	assert.Equal(t, "disk full", entries[0].ErrorMessage)
	assert.Empty(t, entries[0].DownloadURL)
}
