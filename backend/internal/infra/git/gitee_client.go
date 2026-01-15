package git

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// GiteeProvider implements Provider interface for Gitee
type GiteeProvider struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

// NewGiteeProvider creates a new Gitee provider
func NewGiteeProvider(baseURL, accessToken string) (*GiteeProvider, error) {
	if baseURL == "" {
		baseURL = "https://gitee.com/api/v5"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &GiteeProvider{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *GiteeProvider) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := p.baseURL + path

	// Add access_token to URL for Gitee
	if strings.Contains(reqURL, "?") {
		reqURL += "&access_token=" + p.accessToken
	} else {
		reqURL += "?access_token=" + p.accessToken
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

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
