package sso

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCConfig holds OIDC provider configuration
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// OIDCProvider implements Provider for OpenID Connect
type OIDCProvider struct {
	config   *OIDCConfig
	provider *oidc.Provider
	oauth2   oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(ctx context.Context, cfg *OIDCConfig) (*OIDCProvider, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" {
		return nil, fmt.Errorf("%w: missing OIDC issuer URL or client ID", ErrInvalidConfig)
	}
	// ClientSecret is optional: public clients (PKCE) don't require it.
	// If the IdP requires a secret, the code exchange will fail with a clear error.

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}

	oauth2Cfg := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	return &OIDCProvider{
		config:   cfg,
		provider: provider,
		oauth2:   oauth2Cfg,
		verifier: verifier,
	}, nil
}

// GetAuthURL returns the OIDC authorization URL
func (p *OIDCProvider) GetAuthURL(_ context.Context, state string) (string, error) {
	return p.oauth2.AuthCodeURL(state), nil
}

// HandleCallback exchanges the authorization code for tokens and returns user info
func (p *OIDCProvider) HandleCallback(ctx context.Context, params map[string]string) (*UserInfo, error) {
	code := params["code"]
	if code == "" {
		return nil, fmt.Errorf("%w: missing authorization code", ErrAuthFailed)
	}

	// Exchange code for token
	token, err := p.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("%w: no id_token in response", ErrAuthFailed)
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims
	var claims struct {
		Sub      string `json:"sub"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Username string `json:"preferred_username"`
		Picture  string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	if claims.Sub == "" {
		return nil, fmt.Errorf("%w: sub claim is empty in ID token", ErrAuthFailed)
	}
	if claims.Email == "" {
		return nil, fmt.Errorf("%w: email claim is empty in ID token", ErrAuthFailed)
	}

	return &UserInfo{
		ExternalID: claims.Sub,
		Email:      claims.Email,
		Username:   claims.Username,
		Name:       claims.Name,
		AvatarURL:  claims.Picture,
	}, nil
}

// Authenticate is not supported for OIDC
func (p *OIDCProvider) Authenticate(_ context.Context, _, _ string) (*UserInfo, error) {
	return nil, ErrNotSupported
}
