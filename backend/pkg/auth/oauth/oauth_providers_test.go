package oauth

import (
	"strings"
	"testing"
)

// Provider parsers raw-passthrough fields; the user_oauth.go funnel then
// runs EnsureUniqueUsername to coerce them. These tests pin the contract
// that providers DON'T attempt their own sanitize — that responsibility is
// owned by the helper, not by each provider implementation.

func TestParseGoogleUserInfo_NoUsernameAtBoundary(t *testing.T) {
	body := []byte(`{
		"id": "1234567890",
		"email": "kudin.private@gmail.com",
		"name": "Roman Kudin",
		"picture": "https://example.com/avatar.png",
		"verified_email": true
	}`)

	info, err := parseGoogleUserInfo(body)
	if err != nil {
		t.Fatalf("parseGoogleUserInfo: %v", err)
	}
	if info.Username != "" {
		t.Errorf("Google should not derive username at parse boundary; got %q", info.Username)
	}
	if info.Email != "kudin.private@gmail.com" {
		t.Errorf("email round-trip failed: %q", info.Email)
	}
}

func TestParseGoogleUserInfo_UnverifiedEmailRejected(t *testing.T) {
	body := []byte(`{
		"id": "1",
		"email": "x@y.com",
		"verified_email": false
	}`)
	if _, err := parseGoogleUserInfo(body); err == nil {
		t.Fatal("expected error for unverified email")
	}
}

func TestParseGitLabUserInfo_PassesThroughDotUsername(t *testing.T) {
	// GitLab's username regex allows dots; we MUST pass through so
	// user_oauth.go can decide on the sanitized form.
	body := []byte(`{
		"id": 42,
		"username": "alice.bob",
		"email": "alice@example.com",
		"name": "Alice"
	}`)

	info, err := parseGitLabUserInfo(body)
	if err != nil {
		t.Fatalf("parseGitLabUserInfo: %v", err)
	}
	if info.Username != "alice.bob" {
		t.Errorf("expected raw passthrough, got %q", info.Username)
	}
}

func TestParseGitHubUserInfo_PassesLogin(t *testing.T) {
	body := []byte(`{"id": 7, "login": "octo-cat", "email": "o@e.com"}`)
	info, err := parseGitHubUserInfo(body)
	if err != nil {
		t.Fatalf("parseGitHubUserInfo: %v", err)
	}
	if info.Username != "octo-cat" {
		t.Errorf("expected octo-cat, got %q", info.Username)
	}
}

func TestParseGiteeUserInfo_PassesLogin(t *testing.T) {
	body := []byte(`{"id": 7, "login": "gitee_user", "email": "g@e.com"}`)
	info, err := parseGiteeUserInfo(body)
	if err != nil {
		t.Fatalf("parseGiteeUserInfo: %v", err)
	}
	// Gitee logins may contain underscores — sanctioned raw passthrough.
	if !strings.Contains(info.Username, "gitee") {
		t.Errorf("got %q", info.Username)
	}
}
