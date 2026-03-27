package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Token represents OAuth tokens
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    time.Time
}

// UserInfo represents user info from OAuth provider
type UserInfo struct {
	ID        string
	Username  string
	Email     string
	Name      string
	AvatarURL string
}

// httpClient is a shared HTTP client with a reasonable timeout for OAuth calls.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// Config represents OAuth provider configuration
type Config struct {
	Provider        string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	AuthEndpoint    string
	TokenEndpoint   string
	UserInfoURL     string
	Scopes          []string
	BaseURL         string // For self-hosted providers
}

// AuthURL generates the authorization URL
func (c *Config) AuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.ClientID)
	params.Set("redirect_uri", c.RedirectURL)
	params.Set("response_type", "code")
	params.Set("state", state)
	if len(c.Scopes) > 0 {
		params.Set("scope", strings.Join(c.Scopes, " "))
	}
	return c.AuthEndpoint + "?" + params.Encode()
}

// Exchange exchanges the authorization code for tokens
func (c *Config) Exchange(ctx context.Context, code string) (*Token, error) {
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("client_secret", c.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", c.RedirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", c.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	token := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
	}
	if tokenResp.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return token, nil
}

// GetUserInfo fetches user information from the OAuth provider
func (c *Config) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed: %s", string(body))
	}

	userInfo, err := c.parseUserInfo(body)
	if err != nil {
		return nil, err
	}

	// GitHub users may have a private email; fall back to /user/emails API
	if c.Provider == "github" && userInfo.Email == "" {
		userInfo.Email, _ = fetchGitHubPrimaryEmail(ctx, accessToken)
	}

	return userInfo, nil
}

// parseUserInfo parses user info response based on provider
func (c *Config) parseUserInfo(body []byte) (*UserInfo, error) {
	switch c.Provider {
	case "github":
		return parseGitHubUserInfo(body)
	case "google":
		return parseGoogleUserInfo(body)
	case "gitlab":
		return parseGitLabUserInfo(body)
	case "gitee":
		return parseGiteeUserInfo(body)
	default:
		return nil, fmt.Errorf("unknown provider: %s", c.Provider)
	}
}

// NewGitHubConfig creates a GitHub OAuth configuration
// Includes "repo" scope for repository access (browsing and cloning)
func NewGitHubConfig(clientID, clientSecret, redirectURL string) *Config {
	return &Config{
		Provider:      "github",
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		RedirectURL:   redirectURL,
		AuthEndpoint:  "https://github.com/login/oauth/authorize",
		TokenEndpoint: "https://github.com/login/oauth/access_token",
		UserInfoURL:   "https://api.github.com/user",
		Scopes:        []string{"user:email", "read:user", "repo"},
	}
}

// NewGoogleConfig creates a Google OAuth configuration
func NewGoogleConfig(clientID, clientSecret, redirectURL string) *Config {
	return &Config{
		Provider:      "google",
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		RedirectURL:   redirectURL,
		AuthEndpoint:  "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint: "https://oauth2.googleapis.com/token",
		UserInfoURL:   "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes:        []string{"openid", "email", "profile"},
	}
}

// NewGitLabConfig creates a GitLab OAuth configuration
// Includes repository scopes for browsing and cloning
func NewGitLabConfig(clientID, clientSecret, redirectURL, baseURL string) *Config {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return &Config{
		Provider:      "gitlab",
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		RedirectURL:   redirectURL,
		AuthEndpoint:  baseURL + "/oauth/authorize",
		TokenEndpoint: baseURL + "/oauth/token",
		UserInfoURL:   baseURL + "/api/v4/user",
		Scopes:        []string{"read_user", "openid", "profile", "email", "read_repository", "write_repository", "read_api"},
		BaseURL:       baseURL,
	}
}

// NewGiteeConfig creates a Gitee OAuth configuration
// Includes "projects" scope for repository access
func NewGiteeConfig(clientID, clientSecret, redirectURL string) *Config {
	return &Config{
		Provider:      "gitee",
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		RedirectURL:   redirectURL,
		AuthEndpoint:  "https://gitee.com/oauth/authorize",
		TokenEndpoint: "https://gitee.com/oauth/token",
		UserInfoURL:   "https://gitee.com/api/v5/user",
		Scopes:        []string{"user_info", "emails", "projects"},
	}
}

