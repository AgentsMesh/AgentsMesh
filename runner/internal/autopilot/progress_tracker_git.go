// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// getGitDiffSummary retrieves git diff information for the working directory.
func (pt *ProgressTracker) getGitDiffSummary() *GitDiffSummary {
	summary := &GitDiffSummary{
		FilesChanged:   make([]string, 0),
		UnstagedFiles:  make([]string, 0),
		StagedFiles:    make([]string, 0),
		UntrackedFiles: make([]string, 0),
	}

	// Check if directory exists
	if _, err := os.Stat(pt.workDir); os.IsNotExist(err) {
		return summary
	}

	// Check if it's a git repository
	gitDir := filepath.Join(pt.workDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Not a git repository, return empty summary
		return summary
	}

	// Get git status (porcelain format for easy parsing)
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = pt.workDir
	output, err := cmd.Output()
	if err != nil {
		logger.AutopilotTrace().Trace("Failed to get git status", "error", err)
		return summary
	}

	// Parse git status output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := line[:2]
		file := strings.TrimSpace(line[3:])

		if file == "" {
			continue
		}

		switch {
		case status == "??":
			summary.UntrackedFiles = append(summary.UntrackedFiles, file)
		case status[0] != ' ':
			summary.StagedFiles = append(summary.StagedFiles, file)
		case status[1] != ' ':
			summary.UnstagedFiles = append(summary.UnstagedFiles, file)
		}

		summary.FilesChanged = append(summary.FilesChanged, file)
	}

	// Get diff stats
	cmd = exec.Command("git", "diff", "--stat")
	cmd.Dir = pt.workDir
	output, err = cmd.Output()
	if err == nil {
		pt.parseDiffStats(string(output), summary)
	}

	// Also include staged changes
	cmd = exec.Command("git", "diff", "--cached", "--stat")
	cmd.Dir = pt.workDir
	output, err = cmd.Output()
	if err == nil {
		stagedSummary := &GitDiffSummary{}
		pt.parseDiffStats(string(output), stagedSummary)
		summary.Insertions += stagedSummary.Insertions
		summary.Deletions += stagedSummary.Deletions
	}

	summary.HasChanges = len(summary.FilesChanged) > 0

	return summary
}

// parseDiffStats parses the git diff --stat output.
func (pt *ProgressTracker) parseDiffStats(output string, summary *GitDiffSummary) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for summary line: " N files changed, X insertions(+), Y deletions(-)"
		// Also handles singular forms: "1 insertion(+)", "1 deletion(-)"
		hasInsertions := strings.Contains(line, "insertion(+)") || strings.Contains(line, "insertions(+)")
		hasDeletions := strings.Contains(line, "deletion(-)") || strings.Contains(line, "deletions(-)")

		if hasInsertions || hasDeletions {
			// Parse insertions (handles both singular and plural)
			for _, marker := range []string{"insertions(+)", "insertion(+)"} {
				if idx := strings.Index(line, marker); idx > 0 {
					parts := strings.Fields(line[:idx])
					if len(parts) >= 1 {
						fmt.Sscanf(parts[len(parts)-1], "%d", &summary.Insertions)
					}
					break
				}
			}

			// Parse deletions (handles both singular and plural)
			for _, marker := range []string{"deletions(-)", "deletion(-)"} {
				if idx := strings.Index(line, marker); idx > 0 {
					parts := strings.Fields(line[:idx])
					if len(parts) >= 1 {
						fmt.Sscanf(parts[len(parts)-1], "%d", &summary.Deletions)
					}
					break
				}
			}
		}
	}
}

// generateContentHash generates a hash of the current state for change detection.
func (pt *ProgressTracker) generateContentHash(diff *GitDiffSummary) string {
	hasher := sha256.New()

	// Include file list
	for _, f := range diff.FilesChanged {
		hasher.Write([]byte(f))
	}

	// Include stats
	hasher.Write([]byte(fmt.Sprintf("%d:%d", diff.Insertions, diff.Deletions)))

	// Include untracked files
	for _, f := range diff.UntrackedFiles {
		hasher.Write([]byte(f))
	}

	return hex.EncodeToString(hasher.Sum(nil))[:16]
}
