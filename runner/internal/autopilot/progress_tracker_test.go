package autopilot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgressTracker(t *testing.T) {
	workDir := filepath.Join(os.TempDir(), "test")
	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: workDir,
	})

	assert.NotNil(t, pt)
	assert.Equal(t, workDir, pt.workDir)
	assert.Empty(t, pt.snapshots)
	assert.Nil(t, pt.lastSnapshot)
}

func TestProgressTracker_CaptureSnapshot_NonGitDir(t *testing.T) {
	// Create a temporary non-git directory
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	snapshot := pt.CaptureSnapshot()

	assert.NotNil(t, snapshot)
	assert.Empty(t, snapshot.FilesModified)
	assert.NotNil(t, snapshot.GitDiff)
	assert.False(t, snapshot.GitDiff.HasChanges)
}

func TestProgressTracker_CaptureSnapshot_GitDir(t *testing.T) {
	// Create a temporary git repository
	tmpDir, err := os.MkdirTemp("", "progress-test-git-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git user for commit
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	// Initial snapshot (no changes)
	snapshot := pt.CaptureSnapshot()
	assert.NotNil(t, snapshot)
	assert.Empty(t, snapshot.FilesModified)

	// Create a new file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello world"), 0644)
	require.NoError(t, err)

	// Capture another snapshot (should see untracked file)
	snapshot = pt.CaptureSnapshot()
	assert.NotNil(t, snapshot)
	assert.Contains(t, snapshot.FilesModified, "test.txt")
	assert.Contains(t, snapshot.GitDiff.UntrackedFiles, "test.txt")
	assert.True(t, snapshot.GitDiff.HasChanges)
}

func TestProgressTracker_HasProgress(t *testing.T) {
	// Create a temporary git repository
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	// No snapshot yet
	assert.False(t, pt.HasProgress())

	// Take initial snapshot
	pt.CaptureSnapshot()
	assert.False(t, pt.HasProgress()) // No changes since snapshot

	// Create a new file
	testFile := filepath.Join(tmpDir, "newfile.txt")
	err = os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Should detect progress
	assert.True(t, pt.HasProgress())
}

func TestProgressTracker_IsStuck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	// Not stuck with no snapshots
	assert.False(t, pt.IsStuck(1*time.Second))

	// Take multiple snapshots with same state
	for i := 0; i < 5; i++ {
		pt.CaptureSnapshot()
		time.Sleep(10 * time.Millisecond)
	}

	// Should be stuck with very short threshold
	assert.True(t, pt.IsStuck(1*time.Millisecond))

	// Should not be stuck with long threshold
	assert.False(t, pt.IsStuck(1*time.Hour))
}

func TestProgressTracker_GenerateSummary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	// No snapshot
	summary := pt.GenerateSummary()
	assert.Equal(t, "No progress data available", summary)

	// Take a snapshot
	pt.CaptureSnapshot()
	summary = pt.GenerateSummary()
	assert.Contains(t, summary, "No file changes detected")
	assert.Contains(t, summary, "1 snapshot(s) captured")
}

func TestProgressTracker_Reset(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	pt.CaptureSnapshot()
	pt.CaptureSnapshot()

	assert.Equal(t, 2, pt.GetSnapshotCount())
	assert.NotNil(t, pt.GetLastSnapshot())

	pt.Reset()

	assert.Equal(t, 0, pt.GetSnapshotCount())
	assert.Nil(t, pt.GetLastSnapshot())
}

func TestProgressTracker_GetChangedFilesSince(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	// Take initial snapshot
	startTime := time.Now()
	pt.CaptureSnapshot()

	// Create files
	for _, name := range []string{"file1.txt", "file2.txt"} {
		err = os.WriteFile(filepath.Join(tmpDir, name), []byte("content"), 0644)
		require.NoError(t, err)
	}

	time.Sleep(10 * time.Millisecond)

	// Take another snapshot
	pt.CaptureSnapshot()

	// Get files changed since start
	changedFiles := pt.GetChangedFilesSince(startTime)

	// Should include the new files
	assert.Contains(t, changedFiles, "file1.txt")
	assert.Contains(t, changedFiles, "file2.txt")
}

func TestProgressTracker_NonExistentDir(t *testing.T) {
	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: filepath.Join(os.TempDir(), "nonexistent_path_that_does_not_exist"),
	})

	snapshot := pt.CaptureSnapshot()

	assert.NotNil(t, snapshot)
	assert.Empty(t, snapshot.FilesModified)
	assert.NotNil(t, snapshot.GitDiff)
	assert.False(t, snapshot.GitDiff.HasChanges)
}

func TestGitDiffSummary_ParseDiffStats(t *testing.T) {
	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: os.TempDir(),
	})

	tests := []struct {
		name           string
		output         string
		wantInsertions int
		wantDeletions  int
	}{
		{
			name:           "multiple files",
			output:         " file.go | 10 +++++++---\n 3 files changed, 15 insertions(+), 5 deletions(-)",
			wantInsertions: 15,
			wantDeletions:  5,
		},
		{
			name:           "single insertion",
			output:         " 1 file changed, 1 insertion(+)",
			wantInsertions: 1,
			wantDeletions:  0,
		},
		{
			name:           "single deletion",
			output:         " 1 file changed, 1 deletion(-)",
			wantInsertions: 0,
			wantDeletions:  1,
		},
		{
			name:           "empty output",
			output:         "",
			wantInsertions: 0,
			wantDeletions:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := &GitDiffSummary{}
			pt.parseDiffStats(tt.output, summary)

			assert.Equal(t, tt.wantInsertions, summary.Insertions)
			assert.Equal(t, tt.wantDeletions, summary.Deletions)
		})
	}
}

func TestProgressTracker_ConcurrentAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir: tmpDir,
	})

	// Run concurrent operations
	done := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			pt.CaptureSnapshot()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = pt.HasProgress()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = pt.GenerateSummary()
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// Should complete without race conditions
	assert.True(t, pt.GetSnapshotCount() >= 0)
}
