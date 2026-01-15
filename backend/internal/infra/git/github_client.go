package git

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubProvider implements Provider interface for GitHub
type GitHubProvider struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

// NewGitHubProvider creates a new GitHub provider
func NewGitHubProvider(baseURL, accessToken string) (*GitHubProvider, error) {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &GitHubProvider{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *GitHubProvider) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := p.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, ErrNotFound
	}
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, ErrRateLimited
	}

	return resp, nil
}
