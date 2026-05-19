package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type GitLabProvider struct {
	config  *OAuthConfig
	baseURL string
}

func NewGitLabProvider(config *OAuthConfig, baseURL string) *GitLabProvider {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return &GitLabProvider{config: config, baseURL: baseURL}
}

func (p *GitLabProvider) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {p.config.ClientID},
		"redirect_uri":  {p.config.RedirectURL},
		"response_type": {"code"},
		"scope":         {strings.Join(p.config.Scopes, " ")},
		"state":         {state},
	}
	return p.baseURL + "/oauth/authorize?" + params.Encode()
}

func (p *GitLabProvider) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{
		"client_id":     {p.config.ClientID},
		"client_secret": {p.config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {p.config.RedirectURL},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (p *GitLabProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/v4/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var gitlabUser struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &gitlabUser); err != nil {
		return nil, err
	}

	return &OAuthUserInfo{
		ID:        fmt.Sprintf("%d", gitlabUser.ID),
		Username:  gitlabUser.Username,
		Email:     gitlabUser.Email,
		Name:      gitlabUser.Name,
		AvatarURL: gitlabUser.AvatarURL,
	}, nil
}
