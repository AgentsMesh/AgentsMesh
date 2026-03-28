// Package updater provides self-update functionality for the runner.
// It uses GitHub Releases from AgentsMesh/AgentsMesh to download and install updates.
package updater

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	// RepoOwner is the GitHub organization/user that owns the runner repository.
	RepoOwner = "AgentsMesh"
	// RepoName is the name of the runner repository on GitHub.
	RepoName = "AgentsMesh"
)

// UpdateInfo contains information about an available update.
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	PublishedAt    time.Time
	HasUpdate      bool
	AssetURL       string
	AssetName      string
}

// Updater handles checking for and applying updates.
type Updater struct {
	currentVersion  string
	allowPrerelease bool
	detector        ReleaseDetector
	execPathFunc    func() (string, error) // For testing
}

// Option configures the Updater.
type Option func(*Updater)

// WithPrerelease allows updating to prerelease versions.
func WithPrerelease(allow bool) Option {
	return func(u *Updater) {
		u.allowPrerelease = allow
	}
}

// WithReleaseDetector sets a custom release detector (for testing).
func WithReleaseDetector(detector ReleaseDetector) Option {
	return func(u *Updater) {
		u.detector = detector
	}
}

// WithExecPathFunc sets a custom function to get executable path (for testing).
func WithExecPathFunc(f func() (string, error)) Option {
	return func(u *Updater) {
		u.execPathFunc = f
	}
}

// New creates a new Updater instance.
func New(version string, opts ...Option) *Updater {
	u := &Updater{
		currentVersion:  normalizeVersion(version),
		allowPrerelease: false,
		execPathFunc:    os.Executable,
	}

	for _, opt := range opts {
		opt(u)
	}

	return u
}

// normalizeVersion ensures the version has a 'v' prefix for semver comparison.
func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "dev" {
		return "v0.0.0-dev"
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return version
}

// getDetector returns the release detector, creating one if needed.
func (u *Updater) getDetector() (ReleaseDetector, error) {
	if u.detector != nil {
		return u.detector, nil
	}

	detector, err := NewGitHubReleaseDetector()
	if err != nil {
		return nil, err
	}
	u.detector = detector
	return detector, nil
}

// CheckForUpdate checks if a newer version is available.
func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	detector, err := u.getDetector()
	if err != nil {
		slog.Error("Failed to create release detector", "error", err)
		return nil, err
	}

	// Parse current version
	currentSemver, err := semver.NewVersion(u.currentVersion)
	if err != nil {
		// If version parsing fails (e.g., "dev"), treat as v0.0.0
		currentSemver, _ = semver.NewVersion("0.0.0")
	}

	// Find latest release
	release, found, err := detector.DetectLatest(ctx)
	if err != nil {
		slog.Error("Failed to detect latest version", "repo", RepoOwner+"/"+RepoName, "error", err)
		return nil, fmt.Errorf("failed to detect latest version from %s/%s: %w", RepoOwner, RepoName, err)
	}

	if !found {
		return &UpdateInfo{
			CurrentVersion: u.currentVersion,
			HasUpdate:      false,
		}, nil
	}

	// Parse latest version
	latestSemver, err := semver.NewVersion(release.Version)
	if err != nil {
		slog.Error("Failed to parse latest version", "version", release.Version, "error", err)
		return nil, fmt.Errorf("failed to parse latest version %q: %w", release.Version, err)
	}

	// Check if update is available
	hasUpdate := latestSemver.GreaterThan(currentSemver)

	// Filter out prereleases if not allowed
	if !u.allowPrerelease && latestSemver.Prerelease() != "" {
		hasUpdate = false
	}

	return &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.Version,
		ReleaseNotes:   release.ReleaseNotes,
		PublishedAt:    release.PublishedAt,
		HasUpdate:      hasUpdate,
		AssetURL:       release.AssetURL,
		AssetName:      release.AssetName,
	}, nil
}

// UpdateNow checks for updates and applies them immediately.
// The detector's UpdateBinary replaces the executable at execPath in-place
// using go-selfupdate's atomic rename dance.
func (u *Updater) UpdateNow(ctx context.Context) (string, error) {
	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		return "", err
	}

	if !info.HasUpdate {
		return "", nil
	}

	if err := u.updateBinary(ctx, info.LatestVersion); err != nil {
		return "", err
	}

	return info.LatestVersion, nil
}

// UpdateToVersion updates to a specific version.
func (u *Updater) UpdateToVersion(ctx context.Context, version string) error {
	version = normalizeVersion(version)
	return u.updateBinary(ctx, version)
}

// updateBinary downloads and replaces the current executable with the given version.
func (u *Updater) updateBinary(ctx context.Context, version string) error {
	detector, err := u.getDetector()
	if err != nil {
		slog.Error("Failed to create release detector for update", "error", err)
		return err
	}

	execPath, err := u.execPathFunc()
	if err != nil {
		slog.Error("Failed to get executable path for update", "error", err)
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	slog.Info("Updating binary", "version", version, "path", execPath)
	release := &ReleaseInfo{Version: version}
	if err := detector.UpdateBinary(ctx, release, execPath); err != nil {
		slog.Error("Failed to update binary", "version", version, "error", err)
		return fmt.Errorf("failed to update binary: %w", err)
	}

	slog.Info("Binary updated successfully", "version", version)
	return nil
}

// CurrentVersion returns the current version.
func (u *Updater) CurrentVersion() string {
	return u.currentVersion
}
