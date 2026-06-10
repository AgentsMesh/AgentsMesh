package config

type OAuthConfig struct {
	DefaultRedirectURL string // Redirect path after OAuth (e.g., "/")
	DisableSignup      bool   // When true, OAuth login only authenticates existing users; no new account is created
	GitHub             OAuthProviderConfig
	Google             OAuthProviderConfig
	GitLab             GitLabOAuthConfig
	Gitee              OAuthProviderConfig
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
}

type GitLabOAuthConfig struct {
	ClientID     string
	ClientSecret string
	BaseURL      string // GitLab server base URL (default: https://gitlab.com)
}
