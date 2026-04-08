package runnerlog

import (
	"context"
	"io"
	"testing"
	"time"

	runnerlogDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runnerlog"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/infra/storage"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage implements storage.Storage for tests.
type mockStorage struct{}

var _ storage.Storage = (*mockStorage)(nil)

func (m *mockStorage) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) (*storage.FileInfo, error) {
	return &storage.FileInfo{}, nil
}

func (m *mockStorage) Delete(_ context.Context, _ string) error { return nil }

func (m *mockStorage) GetURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://mock-s3.example.com/download", nil
}

func (m *mockStorage) GetInternalURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://mock-s3-internal.example.com/download", nil
}

func (m *mockStorage) PresignPutURL(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return "https://mock-s3.example.com/presigned-put", nil
}

func (m *mockStorage) InternalPresignPutURL(_ context.Context, _, _ string, _ time.Duration) (string, error) {
	return "https://mock-s3-internal.example.com/presigned-put", nil
}

func (m *mockStorage) Exists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func setupRunnerLogService(t *testing.T) (*Service, context.Context, int64, int64) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewRunnerLogRepository(db)
	svc := NewService(repo, &mockStorage{})

	userID := testkit.CreateUser(t, db, "admin@test.com", "admin")
	orgID := testkit.CreateOrg(t, db, "test-org", userID)
	runnerID := testkit.CreateRunner(t, db, orgID, "runner-node-1")

	return svc, context.Background(), orgID, runnerID
}

func TestRunnerLog_CreateAndQuery(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.NotEmpty(t, req.RequestID)
	assert.NotEmpty(t, req.PresignedURL)
	assert.True(t, req.ExpiresAt > time.Now().Unix())

	// List should return the newly created record
	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, req.RequestID, entries[0].RequestID)
	assert.Equal(t, runnerlogDomain.StatusPending, entries[0].Status)
	assert.Empty(t, entries[0].DownloadURL) // pending — no download URL
}

func TestRunnerLog_StatusTransition(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)

	// Transition: pending -> collecting
	svc.HandleUploadStatus(runnerID, req.RequestID, runnerlogDomain.StatusCollecting, 0, "", "", 0)

	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, runnerlogDomain.StatusCollecting, entries[0].Status)

	// Transition: collecting -> uploading
	svc.HandleUploadStatus(runnerID, req.RequestID, runnerlogDomain.StatusUploading, 50, "uploading", "", 0)

	entries, err = svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, runnerlogDomain.StatusUploading, entries[0].Status)

	// Transition: uploading -> completed
	svc.HandleUploadStatus(runnerID, req.RequestID, runnerlogDomain.StatusCompleted, 100, "done", "", 1024)

	entries, err = svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, runnerlogDomain.StatusCompleted, entries[0].Status)
	assert.Equal(t, int64(1024), entries[0].SizeBytes)
	assert.NotEmpty(t, entries[0].DownloadURL) // completed — download URL present
}

func TestRunnerLog_MarkFailed(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)

	svc.MarkFailed(req.RequestID, "gRPC send error")

	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, runnerlogDomain.StatusFailed, entries[0].Status)
	assert.Equal(t, "gRPC send error", entries[0].ErrorMessage)
}

func TestRunnerLog_TerminalStatusCannotBeOverwritten(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)

	// Complete the record
	svc.HandleUploadStatus(runnerID, req.RequestID, runnerlogDomain.StatusCompleted, 100, "", "", 512)

	// Attempting to transition back should be silently rejected
	svc.HandleUploadStatus(runnerID, req.RequestID, runnerlogDomain.StatusUploading, 50, "", "", 0)

	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, runnerlogDomain.StatusCompleted, entries[0].Status) // still completed
}

func TestRunnerLog_InvalidPhaseIgnored(t *testing.T) {
	svc, ctx, orgID, runnerID := setupRunnerLogService(t)

	req, err := svc.RequestUpload(ctx, orgID, runnerID, 1)
	require.NoError(t, err)

	// An invalid phase should be silently ignored
	svc.HandleUploadStatus(runnerID, req.RequestID, "bogus_phase", 0, "", "", 0)

	entries, err := svc.ListByRunner(ctx, orgID, runnerID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, runnerlogDomain.StatusPending, entries[0].Status)
}
