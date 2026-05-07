package v1

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
)

// Tests AuthHandler.isAllowedRedirect across the four redirect classes:
//   - same-host https (web SPA callback)
//   - relative path (in-app navigation)
//   - agentsmesh:// deep link (Electron desktop OAuth callback)
//   - everything else (must be rejected)
func TestIsAllowedRedirect(t *testing.T) {
	h := &AuthHandler{config: &config.Config{PrimaryDomain: "agentsmesh.ai"}}

	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"same-host https", "https://agentsmesh.ai/auth/callback", true},
		{"same-host with port", "https://agentsmesh.ai:443/auth/callback", true},
		{"relative path", "/auth/callback", true},
		{"protocol-relative", "//evil.com/auth/callback", false},
		{"different host", "https://evil.com/auth/callback", false},
		{"http on prod host", "http://agentsmesh.ai/auth/callback", true},
		{"desktop deep link", "agentsmesh://oauth/callback", true},
		{"desktop deep link with query", "agentsmesh://oauth/callback?token=abc", true},
		{"deep link wrong host", "agentsmesh://attacker/oauth/callback", false},
		{"deep link wrong path", "agentsmesh://oauth/evil", false},
		{"unknown scheme", "javascript:alert(1)", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := h.isAllowedRedirect(tc.url)
			if got != tc.want {
				t.Errorf("isAllowedRedirect(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}
