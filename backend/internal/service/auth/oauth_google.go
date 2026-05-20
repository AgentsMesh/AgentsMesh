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

func getGoogleAuthURL(cfg OAuthConfig, state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=email profile" +
		"&state=" + state
}

func handleGoogleCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	accessToken, err := exchangeGoogleCode(ctx, cfg, code)
	if err != nil {
		return nil, err
	}

	return fetchGoogleUserInfo(ctx, accessToken)
}

func exchangeGoogleCode(ctx context.Context, cfg OAuthConfig, code string) (string, error) {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	tokenResp, err := client.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		slog.ErrorContext(ctx, "Google OAuth code exchange failed", "error", err)
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		slog.ErrorContext(ctx, "failed to decode Google OAuth token response", "error", err)
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		slog.WarnContext(ctx, "Google OAuth returned invalid code", "oauth_error", tokenData.Error)
		return "", ErrInvalidOAuthCode
	}

	return tokenData.AccessToken, nil
}

func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	userResp, err := client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch Google user info", "error", err)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Google user info API returned non-OK status", "status", userResp.StatusCode)
		return nil, fmt.Errorf("google API returned status %d", userResp.StatusCode)
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&googleUser); err != nil {
		slog.ErrorContext(ctx, "failed to decode Google user info", "error", err)
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Username is left empty here; user_oauth.go derives a slugkit-compliant
	// handle via EnsureUniqueUsername from email/name once user creation runs.
	return &OAuthUserInfo{
		ID:          googleUser.ID,
		Email:       googleUser.Email,
		Name:        googleUser.Name,
		AvatarURL:   googleUser.Picture,
		AccessToken: accessToken,
	}, nil
}
