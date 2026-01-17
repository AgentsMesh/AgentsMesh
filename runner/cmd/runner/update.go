package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

func runUpdate(args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)

	checkOnly := fs.Bool("check", false, "Only check for updates, don't install")
	fs.BoolVar(checkOnly, "c", false, "Only check for updates (shorthand)")

	autoYes := fs.Bool("yes", false, "Skip confirmation prompt, install directly")
	fs.BoolVar(autoYes, "y", false, "Skip confirmation (shorthand)")

	targetVersion := fs.String("version", "", "Update to a specific version (e.g., v1.2.3)")
	fs.StringVar(targetVersion, "v", "", "Target version (shorthand)")

	force := fs.Bool("force", false, "Force update without waiting for pods to finish")
	fs.BoolVar(force, "f", false, "Force update (shorthand)")

	prerelease := fs.Bool("pre", false, "Allow updating to prerelease versions")

	fs.Usage = func() {
		fmt.Println(`Check and install updates for the AgentsMesh Runner.

Usage:
  runner update [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
Examples:
  runner update              # Interactive update
  runner update --check      # Only check for updates
  runner update -y           # Silent update (wait for pods to finish)
  runner update -f           # Force immediate update (may interrupt pods)
  runner update -v v1.2.3    # Update to specific version
  runner update --pre        # Include prerelease versions`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Create updater with options
	opts := []updater.Option{}
	if *prerelease {
		opts = append(opts, updater.WithPrerelease(true))
	}
	u := updater.New(version, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Check for updates
	spinner := updater.NewSpinnerProgress("Checking for updates...")
	spinner.Start()

	info, err := u.CheckForUpdate(ctx)
	spinner.Stop()

	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Current version: %s\n", info.CurrentVersion)

	// If specific version requested, handle differently
	if *targetVersion != "" {
		fmt.Printf("Target version:  %s\n", *targetVersion)
		if !*autoYes && !confirmUpdate() {
			fmt.Println("Update cancelled.")
			return
		}

		if err := performUpdate(ctx, u, *targetVersion, *force); err != nil {
			fmt.Printf("Update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Regular update flow
	if !info.HasUpdate {
		fmt.Printf("Latest version:  %s\n", info.LatestVersion)
		fmt.Println("\n✓ You are running the latest version!")
		return
	}

	fmt.Printf("Latest version:  %s (released %s)\n", info.LatestVersion, formatPublishedAt(info.PublishedAt))

	if *checkOnly {
		fmt.Println("\n🆕 A new version is available!")
		if info.ReleaseNotes != "" {
			fmt.Println("\nRelease notes:")
			fmt.Println(formatReleaseNotes(info.ReleaseNotes))
		}
		return
	}

	// Show release notes
	if info.ReleaseNotes != "" {
		fmt.Println("\nRelease notes:")
		fmt.Println(formatReleaseNotes(info.ReleaseNotes))
	}

	// Confirm update
	if !*autoYes && !confirmUpdate() {
		fmt.Println("Update cancelled.")
		return
	}

	// Perform update
	if err := performUpdate(ctx, u, info.LatestVersion, *force); err != nil {
		fmt.Printf("Update failed: %v\n", err)
		os.Exit(1)
	}
}

func confirmUpdate() bool {
	fmt.Print("\nDo you want to update? [y/N] ")
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func performUpdate(ctx context.Context, u *updater.Updater, targetVersion string, _ bool) error {
	// Note: force parameter is reserved for future use with graceful updates
	// when running pods need to be considered

	// Create backup first
	fmt.Println("\nCreating backup...")
	backupPath, err := u.CreateBackup()
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Printf("Backup created at: %s\n", backupPath)

	// Download with progress
	fmt.Printf("\nDownloading version %s...\n", targetVersion)
	progress := updater.NewConsoleProgress()

	tmpPath, err := u.Download(ctx, targetVersion, progress.Update)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Apply update
	fmt.Println("\nApplying update...")
	if err := u.Apply(tmpPath); err != nil {
		fmt.Println("\nUpdate failed. Attempting rollback...")
		if rbErr := u.Rollback(); rbErr != nil {
			return fmt.Errorf("update failed (%v) and rollback failed (%v)", err, rbErr)
		}
		fmt.Println("Rollback successful.")
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("\n✓ Successfully updated to %s!\n", targetVersion)
	fmt.Println("\nPlease restart the runner to use the new version.")

	return nil
}

func formatPublishedAt(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

func formatReleaseNotes(notes string) string {
	// Indent each line
	lines := strings.Split(notes, "\n")
	var formatted []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			formatted = append(formatted, "  "+line)
		} else {
			formatted = append(formatted, "")
		}
	}

	result := strings.Join(formatted, "\n")

	// Truncate if too long
	maxLen := 1000
	if len(result) > maxLen {
		result = result[:maxLen] + "\n  ... (truncated)"
	}

	return result
}
