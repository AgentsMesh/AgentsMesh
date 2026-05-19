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

func getGiteeAuthURL(cfg OAuthConfig, state string) string {
	return "https://gitee.com/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=user_info" +
		"&state=" + state
}

func handleGiteeCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	accessToken, err := exchangeGiteeCode(ctx, cfg, code)
	if err != nil {
		return nil, err
	}

	return fetchGiteeUserInfo(ctx, accessToken)
}

func exchangeGiteeCode(ctx context.Context, cfg OAuthConfig, code string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}
	tokenResp, err := client.PostForm("https://gitee.com/oauth/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		slog.ErrorContext(ctx, "Gitee OAuth code exchange failed", "error", err)
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		slog.ErrorContext(ctx, "failed to decode Gitee OAuth token response", "error", err)
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		slog.WarnContext(ctx, "Gitee OAuth returned invalid code", "oauth_error", tokenData.Error)
		return "", ErrInvalidOAuthCode
	}

	return tokenData.AccessToken, nil
}

func fetchGiteeUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitee.com/api/v5/user?access_token="+accessToken, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create Gitee user info request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch Gitee user info", "error", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Gitee user info API returned non-OK status", "status", userResp.StatusCode)
		return nil, fmt.Errorf("gitee API returned status %d", userResp.StatusCode)
	}

	var giteeUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&giteeUser); err != nil {
		slog.ErrorContext(ctx, "failed to decode Gitee user info", "error", err)
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &OAuthUserInfo{
		ID:          fmt.Sprintf("%d", giteeUser.ID),
		Username:    giteeUser.Login,
		Email:       giteeUser.Email,
		Name:        giteeUser.Name,
		AvatarURL:   giteeUser.AvatarURL,
		AccessToken: accessToken,
	}, nil
}
