// Package updater provides self-update functionality for the runner.
// It uses GitHub Releases from AgentsMesh/AgentsMesh to download and install updates.
package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// Download downloads the specified version to a temporary file.
func (u *Updater) Download(ctx context.Context, version string, _ func(downloaded, total int64)) (string, error) {
	detector, err := u.getDetector()
	if err != nil {
		return "", err
	}

	release, found, err := detector.DetectVersion(ctx, version)
	if err != nil {
		return "", fmt.Errorf("failed to find version %s: %w", version, err)
	}
	if !found {
		return "", fmt.Errorf("version %s not found", version)
	}

	// Create temp file in the same directory as the executable to ensure
	// os.Rename works (avoids cross-device link errors on Windows).
	// Fall back to system temp dir if the executable's directory is not writable
	// (e.g., C:\Program Files\ requires admin privileges).
	execPath, err := u.execPathFunc()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(execPath), "runner-update-*")
	if err != nil {
		tmpFile, err = os.CreateTemp("", "runner-update-*")
	}
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	if err := detector.DownloadTo(ctx, release, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to download update: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			return "", fmt.Errorf("failed to set executable permission: %w", err)
		}
	}

	return tmpPath, nil
}

// Apply replaces the current executable with the downloaded update.
func (u *Updater) Apply(tmpPath string) error {
	execPath, err := u.execPathFunc()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if err := atomicReplace(tmpPath, execPath); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	return nil
}

// UpdateNow checks for updates and applies them immediately.
func (u *Updater) UpdateNow(ctx context.Context, progress func(downloaded, total int64)) (string, error) {
	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		return "", err
	}

	if !info.HasUpdate {
		return "", nil
	}

	tmpPath, err := u.Download(ctx, info.LatestVersion, progress)
	if err != nil {
		return "", err
	}

	if err := u.Apply(tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}

	return info.LatestVersion, nil
}

// UpdateToVersion updates to a specific version.
func (u *Updater) UpdateToVersion(ctx context.Context, version string, progress func(downloaded, total int64)) error {
	version = normalizeVersion(version)

	tmpPath, err := u.Download(ctx, version, progress)
	if err != nil {
		return err
	}

	if err := u.Apply(tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// CurrentVersion returns the current version.
func (u *Updater) CurrentVersion() string {
	return u.currentVersion
}

// Rollback restores the previous version if a backup exists.
func (u *Updater) Rollback() error {
	execPath, err := u.execPathFunc()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	backupPath := execPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", backupPath)
	}

	if err := atomicReplace(backupPath, execPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// CreateBackup creates a backup of the current executable.
func (u *Updater) CreateBackup() (string, error) {
	execPath, err := u.execPathFunc()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	backupPath := execPath + ".bak"

	if err := copyFile(execPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}
