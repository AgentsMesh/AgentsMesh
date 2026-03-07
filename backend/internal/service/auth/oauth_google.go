package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// getGoogleAuthURL returns Google OAuth authorization URL
func getGoogleAuthURL(cfg OAuthConfig, state string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=email profile" +
		"&state=" + state
}

// handleGoogleCallback exchanges code for token and fetches user info
func handleGoogleCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	accessToken, err := exchangeGoogleCode(ctx, cfg, code)
	if err != nil {
		return nil, err
	}

	return fetchGoogleUserInfo(ctx, accessToken)
}

// exchangeGoogleCode exchanges authorization code for access token
func exchangeGoogleCode(ctx context.Context, cfg OAuthConfig, code string) (string, error) {
	tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenData.Error != "" || tokenData.AccessToken == "" {
		return "", ErrInvalidOAuthCode
	}

	return tokenData.AccessToken, nil
}

// fetchGoogleUserInfo fetches user info from Google API
func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
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
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Generate username from email if not provided
	username := strings.Split(googleUser.Email, "@")[0]

	return &OAuthUserInfo{
		ID:          googleUser.ID,
		Username:    username,
		Email:       googleUser.Email,
		Name:        googleUser.Name,
		AvatarURL:   googleUser.Picture,
		AccessToken: accessToken,
	}, nil
}
