package git

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitLabProvider implements Provider interface for GitLab
type GitLabProvider struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

// NewGitLabProvider creates a new GitLab provider
func NewGitLabProvider(baseURL, accessToken string) (*GitLabProvider, error) {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &GitLabProvider{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *GitLabProvider) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := fmt.Sprintf("%s/api/v4%s", p.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", p.accessToken)
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
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		return nil, ErrRateLimited
	}

	return resp, nil
}
