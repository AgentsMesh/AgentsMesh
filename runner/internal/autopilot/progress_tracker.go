// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ProgressTracker monitors file changes and git status to track task progress.
// It captures snapshots of the working directory state and detects changes.
type ProgressTracker struct {
	workDir      string
	snapshots    []ProgressSnapshot
	lastSnapshot *ProgressSnapshot
	mu           sync.RWMutex
	log          *slog.Logger
	gitExecutor  GitExecutor
}

// ProgressSnapshot represents a point-in-time state of the working directory.
type ProgressSnapshot struct {
	Timestamp     time.Time
	FilesModified []string
	GitDiff       *GitDiffSummary
	ContentHash   string // Hash of terminal content or key files
}

// GitDiffSummary contains summarized git diff information.
type GitDiffSummary struct {
	FilesChanged   []string
	Insertions     int
	Deletions      int
	UnstagedFiles  []string
	StagedFiles    []string
	UntrackedFiles []string
	HasChanges     bool
}

// ProgressTrackerConfig contains configuration for ProgressTracker.
type ProgressTrackerConfig struct {
	WorkDir     string
	Logger      *slog.Logger
	GitExecutor GitExecutor // Optional: defaults to DefaultGitExecutor
}

// NewProgressTracker creates a new ProgressTracker instance.
func NewProgressTracker(cfg ProgressTrackerConfig) *ProgressTracker {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	gitExecutor := cfg.GitExecutor
	if gitExecutor == nil {
		gitExecutor = NewDefaultGitExecutor()
	}

	return &ProgressTracker{
		workDir:     cfg.WorkDir,
		snapshots:   make([]ProgressSnapshot, 0),
		log:         log,
		gitExecutor: gitExecutor,
	}
}

// CaptureSnapshot captures the current state of the working directory.
func (pt *ProgressTracker) CaptureSnapshot() *ProgressSnapshot {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	snapshot := &ProgressSnapshot{
		Timestamp: time.Now(),
	}

	// Get git diff summary
	gitDiff := pt.getGitDiffSummary()
	snapshot.GitDiff = gitDiff
	snapshot.FilesModified = gitDiff.FilesChanged

	// Generate content hash for change detection
	snapshot.ContentHash = pt.generateContentHash(gitDiff)

	// Store snapshot
	pt.snapshots = append(pt.snapshots, *snapshot)
	pt.lastSnapshot = snapshot

	logger.AutopilotTrace().Trace("Captured progress snapshot",
		"files_changed", len(snapshot.FilesModified),
		"has_changes", gitDiff.HasChanges,
		"insertions", gitDiff.Insertions,
		"deletions", gitDiff.Deletions)

	return snapshot
}

// HasProgress checks if there has been progress since the last snapshot.
func (pt *ProgressTracker) HasProgress() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.lastSnapshot == nil {
		return false
	}

	currentDiff := pt.getGitDiffSummary()

	// Check if there are new file changes
	if currentDiff.HasChanges && !pt.lastSnapshot.GitDiff.HasChanges {
		return true
	}

	// Check if different files are modified
	if len(currentDiff.FilesChanged) != len(pt.lastSnapshot.FilesModified) {
		return true
	}

	// Compare file lists
	currentFiles := make(map[string]bool)
	for _, f := range currentDiff.FilesChanged {
		currentFiles[f] = true
	}
	for _, f := range pt.lastSnapshot.FilesModified {
		if !currentFiles[f] {
			return true
		}
	}

	// Check insertions/deletions
	if currentDiff.Insertions != pt.lastSnapshot.GitDiff.Insertions ||
		currentDiff.Deletions != pt.lastSnapshot.GitDiff.Deletions {
		return true
	}

	return false
}

// IsStuck checks if no progress has been made for the specified duration.
func (pt *ProgressTracker) IsStuck(threshold time.Duration) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if len(pt.snapshots) < 2 {
		return false
	}

	// Check if the last N snapshots have the same content hash
	recentSnapshots := pt.snapshots
	if len(recentSnapshots) > 5 {
		recentSnapshots = recentSnapshots[len(recentSnapshots)-5:]
	}

	// If all recent snapshots have the same hash and enough time has passed
	if len(recentSnapshots) >= 2 {
		firstHash := recentSnapshots[0].ContentHash
		allSame := true
		for _, s := range recentSnapshots[1:] {
			if s.ContentHash != firstHash {
				allSame = false
				break
			}
		}

		if allSame {
			duration := recentSnapshots[len(recentSnapshots)-1].Timestamp.Sub(recentSnapshots[0].Timestamp)
			return duration >= threshold
		}
	}

	return false
}

// GenerateSummary generates a human-readable summary of progress.
func (pt *ProgressTracker) GenerateSummary() string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.lastSnapshot == nil {
		return "No progress data available"
	}

	var parts []string

	// Git changes summary
	if pt.lastSnapshot.GitDiff != nil && pt.lastSnapshot.GitDiff.HasChanges {
		parts = append(parts, fmt.Sprintf("%d file(s) modified", len(pt.lastSnapshot.GitDiff.FilesChanged)))
		if pt.lastSnapshot.GitDiff.Insertions > 0 || pt.lastSnapshot.GitDiff.Deletions > 0 {
			parts = append(parts, fmt.Sprintf("+%d/-%d lines", pt.lastSnapshot.GitDiff.Insertions, pt.lastSnapshot.GitDiff.Deletions))
		}
	} else {
		parts = append(parts, "No file changes detected")
	}

	// Untracked files
	if pt.lastSnapshot.GitDiff != nil && len(pt.lastSnapshot.GitDiff.UntrackedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("%d new file(s)", len(pt.lastSnapshot.GitDiff.UntrackedFiles)))
	}

	// Snapshot count
	parts = append(parts, fmt.Sprintf("%d snapshot(s) captured", len(pt.snapshots)))

	return strings.Join(parts, ", ")
}

// GetLastSnapshot returns the most recent snapshot.
func (pt *ProgressTracker) GetLastSnapshot() *ProgressSnapshot {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.lastSnapshot
}

// GetSnapshotCount returns the total number of snapshots.
func (pt *ProgressTracker) GetSnapshotCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return len(pt.snapshots)
}

// Reset clears all captured snapshots.
func (pt *ProgressTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.snapshots = make([]ProgressSnapshot, 0)
	pt.lastSnapshot = nil
}

// GetChangedFilesSince returns files changed since the given time.
func (pt *ProgressTracker) GetChangedFilesSince(since time.Time) []string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	changedFiles := make(map[string]bool)

	for _, snapshot := range pt.snapshots {
		if snapshot.Timestamp.After(since) {
			for _, f := range snapshot.FilesModified {
				changedFiles[f] = true
			}
		}
	}

	result := make([]string, 0, len(changedFiles))
	for f := range changedFiles {
		result = append(result, f)
	}

	return result
}
