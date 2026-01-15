package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// getGitHubAuthURL returns GitHub OAuth authorization URL
func getGitHubAuthURL(cfg OAuthConfig, state string) string {
	return "https://github.com/login/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&scope=user:email" +
		"&state=" + state
}

// handleGitHubCallback exchanges code for token and fetches user info
func handleGitHubCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	accessToken, err := exchangeGitHubCode(ctx, cfg, code)
	if err != nil {
		return nil, err
	}

	return fetchGitHubUserInfo(ctx, accessToken)
}

// exchangeGitHubCode exchanges authorization code for access token
func exchangeGitHubCode(ctx context.Context, cfg OAuthConfig, code string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	tokenResp, err := client.PostForm("https://github.com/login/oauth/access_token", url.Values{
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {cfg.RedirectURL},
	})
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer tokenResp.Body.Close()

	body, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	accessToken := values.Get("access_token")
	if accessToken == "" {
		return "", ErrInvalidOAuthCode
	}

	return accessToken, nil
}

// fetchGitHubUserInfo fetches user info from GitHub API
func fetchGitHubUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", userResp.StatusCode)
	}

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// If email is empty, fetch from emails endpoint (private emails)
	email := ghUser.Email
	if email == "" {
		email, _ = getGitHubPrimaryEmail(ctx, accessToken)
	}

	return &OAuthUserInfo{
		ID:          fmt.Sprintf("%d", ghUser.ID),
		Username:    ghUser.Login,
		Email:       email,
		Name:        ghUser.Name,
		AvatarURL:   ghUser.AvatarURL,
		AccessToken: accessToken,
	}, nil
}

// getGitHubPrimaryEmail fetches primary email from GitHub emails endpoint
func getGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}
