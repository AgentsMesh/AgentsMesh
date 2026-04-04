package agent

import (
	"sort"
	"testing"
	"time"
)

func TestUserAgentCredentialProfileTableName(t *testing.T) {
	p := UserAgentCredentialProfile{}
	if p.TableName() != "user_agent_credential_profiles" {
		t.Errorf("expected 'user_agent_credential_profiles', got %s", p.TableName())
	}
}

// agentfileWithEnv is a helper that returns an AgentFile source with ENV declarations.
func agentfileWithEnv() string {
	return `AGENT "claude"
ENV api_key SECRET OPTIONAL
ENV auth_token SECRET OPTIONAL
ENV base_url TEXT OPTIONAL
`
}

func TestToResponse_SecretFieldsNotExposed(t *testing.T) {
	// secret fields (api_key, auth_token) must appear in ConfiguredFields
	// but NEVER in ConfiguredValues
	pf := agentfileWithEnv()
	profile := &UserAgentCredentialProfile{
		ID: 1,
		UserID:      10,
		AgentSlug: "claude-code",
		Name:        "work",
		CredentialsEncrypted: EncryptedCredentials{
			"api_key":  "sk-secret-key-123",
			"base_url": "https://proxy.example.com",
		},
		Agent: &Agent{
			Slug:          "claude-code",
			Name:          "Claude Code",
			AgentfileSource: &pf,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := profile.ToResponse()

	// ConfiguredFields should contain both stored keys
	sort.Strings(resp.ConfiguredFields)
	if len(resp.ConfiguredFields) != 2 {
		t.Fatalf("expected 2 configured fields, got %d: %v", len(resp.ConfiguredFields), resp.ConfiguredFields)
	}
	if resp.ConfiguredFields[0] != "api_key" || resp.ConfiguredFields[1] != "base_url" {
		t.Errorf("unexpected ConfiguredFields: %v", resp.ConfiguredFields)
	}

	// ConfiguredValues must contain base_url (text) but NOT api_key (secret)
	if resp.ConfiguredValues == nil {
		t.Fatal("expected ConfiguredValues to be non-nil")
	}
	if resp.ConfiguredValues["base_url"] != "https://proxy.example.com" {
		t.Errorf("expected base_url = 'https://proxy.example.com', got %s", resp.ConfiguredValues["base_url"])
	}
	if _, exists := resp.ConfiguredValues["api_key"]; exists {
		t.Error("api_key (secret) must NOT appear in ConfiguredValues")
	}
}

func TestToResponse_AuthTokenNotExposed(t *testing.T) {
	// Verify auth_token (secret) is never exposed in ConfiguredValues
	pf := agentfileWithEnv()
	profile := &UserAgentCredentialProfile{
		ID: 2,
		UserID:      10,
		AgentSlug: "claude-code",
		Name:        "token-config",
		CredentialsEncrypted: EncryptedCredentials{
			"auth_token": "my-secret-token",
			"base_url":   "https://custom.api.com",
		},
		Agent: &Agent{
			Slug:          "claude-code",
			Name:          "Claude Code",
			AgentfileSource: &pf,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := profile.ToResponse()

	if _, exists := resp.ConfiguredValues["auth_token"]; exists {
		t.Error("auth_token (secret) must NOT appear in ConfiguredValues")
	}
	if resp.ConfiguredValues["base_url"] != "https://custom.api.com" {
		t.Errorf("expected base_url in ConfiguredValues, got %v", resp.ConfiguredValues)
	}

	// auth_token must appear in ConfiguredFields
	found := false
	for _, f := range resp.ConfiguredFields {
		if f == "auth_token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("auth_token should appear in ConfiguredFields")
	}
}

func TestToResponse_NoAgent(t *testing.T) {
	// Without Agent loaded, all fields go to ConfiguredFields only
	// (no AgentFile to determine type, so nothing goes to ConfiguredValues)
	profile := &UserAgentCredentialProfile{
		ID:          3,
		UserID:      10,
		AgentSlug: "claude-code",
		Name:        "no-schema",
		CredentialsEncrypted: EncryptedCredentials{
			"api_key":  "sk-key",
			"base_url": "https://example.com",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := profile.ToResponse()

	if len(resp.ConfiguredFields) != 2 {
		t.Errorf("expected 2 configured fields, got %d", len(resp.ConfiguredFields))
	}
	if resp.ConfiguredValues != nil {
		t.Errorf("expected nil ConfiguredValues without Agent, got %v", resp.ConfiguredValues)
	}
}

func TestToResponse_NilCredentials(t *testing.T) {
	profile := &UserAgentCredentialProfile{
		ID:          4,
		UserID:      10,
		AgentSlug: "claude-code",
		Name:        "runner-host",
		IsRunnerHost: true,
		Agent: &Agent{
			Slug: "claude-code",
			Name: "Claude Code",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := profile.ToResponse()

	if resp.ConfiguredFields != nil {
		t.Errorf("expected nil ConfiguredFields, got %v", resp.ConfiguredFields)
	}
	if resp.ConfiguredValues != nil {
		t.Errorf("expected nil ConfiguredValues, got %v", resp.ConfiguredValues)
	}
	if !resp.IsRunnerHost {
		t.Error("expected IsRunnerHost true")
	}
}

func TestToResponse_EmptyTextValueNotExposed(t *testing.T) {
	// Empty text values should NOT appear in ConfiguredValues
	pf := agentfileWithEnv()
	profile := &UserAgentCredentialProfile{
		ID:          5,
		UserID:      10,
		AgentSlug: "claude-code",
		Name:        "empty-url",
		CredentialsEncrypted: EncryptedCredentials{
			"api_key":  "sk-key",
			"base_url": "",
		},
		Agent: &Agent{
			Slug:          "claude-code",
			Name:          "Claude Code",
			AgentfileSource: &pf,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := profile.ToResponse()

	if resp.ConfiguredValues != nil {
		t.Errorf("expected nil ConfiguredValues for empty text value, got %v", resp.ConfiguredValues)
	}
}

func TestToResponse_AgentInfo(t *testing.T) {
	profile := &UserAgentCredentialProfile{
		ID:          6,
		UserID:      10,
		AgentSlug: "claude-code",
		Name:        "test",
		Agent: &Agent{
			Slug: "claude-code",
			Name: "Claude Code",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	resp := profile.ToResponse()

	if resp.AgentName != "Claude Code" {
		t.Errorf("expected AgentName 'Claude Code', got %s", resp.AgentName)
	}
}
