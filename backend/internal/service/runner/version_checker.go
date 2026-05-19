package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	redisKeyLatestVersion = "runner:latest_version"

	defaultCheckInterval = 1 * time.Hour

	redisCacheTTL = 1 * time.Hour
)

const (
	githubRepo = "AgentsMesh/AgentsMesh"
)

type VersionChecker struct {
	redisClient *redis.Client
	interval    time.Duration
	httpClient  *http.Client
	logger      *slog.Logger
}

// NewVersionChecker returns nil when redisClient is nil — disables the feature.
func NewVersionChecker(redisClient *redis.Client) *VersionChecker {
	if redisClient == nil {
		return nil
	}

	return &VersionChecker{
		redisClient: redisClient,
		interval:    defaultCheckInterval,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		logger: slog.Default().With("component", "version_checker"),
	}
}

func (vc *VersionChecker) Start(ctx context.Context) {
	if vc == nil {
		return
	}

	go func() {
		if _, err := vc.checkGitHubRelease(ctx); err != nil {
			vc.logger.Warn("Initial version check failed", "error", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(vc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := vc.checkGitHubRelease(ctx); err != nil {
					vc.logger.Warn("Periodic version check failed", "error", err)
				}
			}
		}
	}()

	vc.logger.Info("Version checker started", "repo", githubRepo, "interval", vc.interval)
}

func (vc *VersionChecker) GetLatestVersion(ctx context.Context) string {
	if vc == nil || vc.redisClient == nil {
		return ""
	}

	val, err := vc.redisClient.Get(ctx, redisKeyLatestVersion).Result()
	if err != nil {
		return ""
	}
	return val
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Draft   bool   `json:"draft"`
}

func (vc *VersionChecker) checkGitHubRelease(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AgentsMesh-Backend")

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	version := NormalizeVersion(release.TagName)
	if version == "" {
		return "", fmt.Errorf("empty version tag from GitHub")
	}

	if vc.redisClient != nil {
		if err := vc.redisClient.Set(ctx, redisKeyLatestVersion, version, redisCacheTTL).Err(); err != nil {
			vc.logger.Warn("Failed to cache latest version in Redis", "error", err)
		}
	}

	vc.logger.Info("Latest runner version updated", "version", version)
	return version, nil
}

func NormalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
