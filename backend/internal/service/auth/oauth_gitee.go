package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// getGiteeAuthURL returns Gitee OAuth authorization URL
func getGiteeAuthURL(cfg OAuthConfig, state string) string {
	return "https://gitee.com/oauth/authorize" +
		"?client_id=" + cfg.ClientID +
		"&redirect_uri=" + cfg.RedirectURL +
		"&response_type=code" +
		"&scope=user_info" +
		"&state=" + state
}

// handleGiteeCallback exchanges code for token and fetches user info
func handleGiteeCallback(ctx context.Context, cfg OAuthConfig, code string) (*OAuthUserInfo, error) {
	accessToken, err := exchangeGiteeCode(ctx, cfg, code)
	if err != nil {
		return nil, err
	}

	return fetchGiteeUserInfo(ctx, accessToken)
}

// exchangeGiteeCode exchanges authorization code for access token
func exchangeGiteeCode(ctx context.Context, cfg OAuthConfig, code string) (string, error) {
	tokenResp, err := http.PostForm("https://gitee.com/oauth/token", url.Values{
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

// fetchGiteeUserInfo fetches user info from Gitee API
func fetchGiteeUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitee.com/api/v5/user?access_token="+accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
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
