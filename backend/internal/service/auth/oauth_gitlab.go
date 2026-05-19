package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func getGitLabAuthURL(cfg OAuthConfig, state string) string {
	return "https://gitlab.com/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=read_user" +
		"&state=" + state
}

func handleGitLabCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	accessToken, err := exchangeGitLabCode(ctx, cfg, code)
	if err != nil {
		return nil, err
	}

	return fetchGitLabUserInfo(ctx, accessToken)
}

func exchangeGitLabCode(ctx context.Context, cfg OAuthConfig, code string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}
	tokenResp, err := client.PostForm("https://gitlab.com/oauth/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		slog.ErrorContext(ctx, "GitLab OAuth code exchange failed", "error", err)
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		slog.ErrorContext(ctx, "failed to decode GitLab OAuth token response", "error", err)
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		slog.WarnContext(ctx, "GitLab OAuth returned invalid code", "oauth_error", tokenData.Error)
		return "", ErrInvalidOAuthCode
	}

	return tokenData.AccessToken, nil
}

func fetchGitLabUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitlab.com/api/v4/user", nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create GitLab user info request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}
	userResp, err := client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch GitLab user info", "error", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "GitLab user info API returned non-OK status", "status", userResp.StatusCode)
		return nil, fmt.Errorf("GitLab API returned status %d", userResp.StatusCode)
	}

	var glUser struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&glUser); err != nil {
		slog.ErrorContext(ctx, "failed to decode GitLab user info", "error", err)
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &OAuthUserInfo{
		ID:          fmt.Sprintf("%d", glUser.ID),
		Username:    glUser.Username,
		Email:       glUser.Email,
		Name:        glUser.Name,
		AvatarURL:   glUser.AvatarURL,
		AccessToken: accessToken,
	}, nil
}
