package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type GiteeProvider struct {
	config *OAuthConfig
}

func NewGiteeProvider(config *OAuthConfig) *GiteeProvider {
	return &GiteeProvider{config: config}
}

func (p *GiteeProvider) GetAuthURL(state string) string {
	params := url.Values{
		"client_id":     {p.config.ClientID},
		"redirect_uri":  {p.config.RedirectURL},
		"response_type": {"code"},
		"scope":         {strings.Join(p.config.Scopes, " ")},
		"state":         {state},
	}
	return "https://gitee.com/oauth/authorize?" + params.Encode()
}

func (p *GiteeProvider) ExchangeCode(ctx context.Context, code string) (*OAuthToken, error) {
	params := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.config.ClientID},
		"redirect_uri":  {p.config.RedirectURL},
		"client_secret": {p.config.ClientSecret},
	}

	tokenURL := "https://gitee.com/oauth/token?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	if err != nil {
		return nil, err
	}

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

func (p *GiteeProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gitee.com/api/v5/user?access_token="+token.AccessToken, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var giteeUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&giteeUser); err != nil {
		return nil, err
	}

	return &OAuthUserInfo{
		ID:        fmt.Sprintf("%d", giteeUser.ID),
		Username:  giteeUser.Login,
		Email:     giteeUser.Email,
		Name:      giteeUser.Name,
		AvatarURL: giteeUser.AvatarURL,
	}, nil
}
