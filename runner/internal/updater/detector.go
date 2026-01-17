package updater

import (
	"context"
	"fmt"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

// ReleaseInfo represents information about a release.
type ReleaseInfo struct {
	Version      string
	ReleaseNotes string
	PublishedAt  time.Time
	AssetURL     string
	AssetName    string
}

// ReleaseDetector abstracts the release detection and download logic.
type ReleaseDetector interface {
	// DetectLatest finds the latest release.
	DetectLatest(ctx context.Context) (*ReleaseInfo, bool, error)
	// DetectVersion finds a specific version.
	DetectVersion(ctx context.Context, version string) (*ReleaseInfo, bool, error)
	// DownloadTo downloads a release to a specified path.
	DownloadTo(ctx context.Context, release *ReleaseInfo, path string) error
}

// GitHubReleaseDetector implements ReleaseDetector using GitHub API.
type GitHubReleaseDetector struct {
	updater *selfupdate.Updater
}

// NewGitHubReleaseDetector creates a new GitHubReleaseDetector.
func NewGitHubReleaseDetector() (*GitHubReleaseDetector, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}

	return &GitHubReleaseDetector{updater: updater}, nil
}

// DetectLatest finds the latest release from GitHub.
func (g *GitHubReleaseDetector) DetectLatest(ctx context.Context) (*ReleaseInfo, bool, error) {
	release, found, err := g.updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(RepoOwner, RepoName))
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return &ReleaseInfo{
		Version:      release.Version(),
		ReleaseNotes: release.ReleaseNotes,
		PublishedAt:  release.PublishedAt,
		AssetURL:     release.AssetURL,
		AssetName:    release.AssetName,
	}, true, nil
}

// DetectVersion finds a specific version from GitHub.
func (g *GitHubReleaseDetector) DetectVersion(ctx context.Context, version string) (*ReleaseInfo, bool, error) {
	release, found, err := g.updater.DetectVersion(ctx, selfupdate.NewRepositorySlug(RepoOwner, RepoName), version)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	return &ReleaseInfo{
		Version:      release.Version(),
		ReleaseNotes: release.ReleaseNotes,
		PublishedAt:  release.PublishedAt,
		AssetURL:     release.AssetURL,
		AssetName:    release.AssetName,
	}, true, nil
}

// DownloadTo downloads a release to the specified path.
func (g *GitHubReleaseDetector) DownloadTo(ctx context.Context, release *ReleaseInfo, path string) error {
	r, found, err := g.updater.DetectVersion(ctx, selfupdate.NewRepositorySlug(RepoOwner, RepoName), release.Version)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("release not found")
	}
	return g.updater.UpdateTo(ctx, r, path)
}
