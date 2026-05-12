package ssoconnect

import (
	"errors"
	"regexp"
	"strings"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	ssov1 "github.com/anthropics/agentsmesh/proto/gen/go/sso/v1"
)

// domainRegexp mirrors auth_sso.go:21 — restricts the domain payload to
// look-alike domain syntax. Server-side defense matches the legacy
// REST validation so any string the frontend forwards (or any
// hand-crafted curl) gets the same rejection.
var domainRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+$`)

// validateDomainFromRequest lowercases + trims + regex-checks the
// `domain` payload field. Returns Connect errors so handlers can
// `return nil, err` directly.
func validateDomainFromRequest(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	if domain == "" {
		return "", connect.NewError(connect.CodeInvalidArgument,
			errors.New("domain is required"))
	}
	if !domainRegexp.MatchString(domain) {
		return "", connect.NewError(connect.CodeInvalidArgument,
			errors.New("invalid domain format"))
	}
	return domain, nil
}

// toProtoDiscoverConfig maps the domain Config to the sanitized
// discover wire shape — same five fields the REST handler emits via
// ToDiscoverResponse (config_response.go:82). Secrets / IdP URLs stay
// server-side.
func toProtoDiscoverConfig(cfg *sso.Config) *ssov1.SSODiscoverConfig {
	if cfg == nil {
		return nil
	}
	return &ssov1.SSODiscoverConfig{
		Domain:     cfg.Domain,
		Name:       cfg.Name,
		Protocol:   string(cfg.Protocol),
		EnforceSso: cfg.EnforceSSO,
	}
}

// toProtoLdapAuthUser maps the GORM-backed User entity to the public
// user info returned alongside the LDAP token pair. Mirrors the gin.H
// `user` block in auth_sso_ldap.go:59 — same four fields. Passwords and
// other sensitive fields stay server-side.
func toProtoLdapAuthUser(u *domainUser.User) *ssov1.LdapAuthUser {
	if u == nil {
		return nil
	}
	out := &ssov1.LdapAuthUser{
		Id:       u.ID,
		Email:    u.Email,
		Username: u.Username,
	}
	if u.Name != nil && *u.Name != "" {
		n := *u.Name
		out.Name = &n
	}
	return out
}
