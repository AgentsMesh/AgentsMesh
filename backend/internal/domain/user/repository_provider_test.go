package user

import (
	"encoding/json"
	"testing"
)

func TestRepositoryProvider_ToResponse_DefaultIsActiveTrue(t *testing.T) {
	p := &RepositoryProvider{
		ID:           1,
		ProviderType: ProviderTypeGitHub,
		Name:         "GitHub",
		BaseURL:      "https://github.com",
		IsActive:     true,
	}
	resp := p.ToResponse()
	if !resp.IsActive {
		t.Fatal("expected IsActive=true to round-trip through ToResponse")
	}
	if resp.HasIdentity || resp.HasBotToken || resp.HasClientID {
		t.Fatalf("expected all has_* flags to be false when no credentials set, got: %+v", resp)
	}
}

func TestRepositoryProvider_ToResponse_IsActiveFalsePropagates(t *testing.T) {
	p := &RepositoryProvider{ID: 1, ProviderType: ProviderTypeGitHub, Name: "GitHub", IsActive: false}
	resp := p.ToResponse()
	if resp.IsActive {
		t.Fatal("expected IsActive=false to propagate through ToResponse")
	}
}

func TestRepositoryProvider_ToResponse_HasIdentityRequiresAccessToken(t *testing.T) {
	id := int64(7)
	p := &RepositoryProvider{ID: 1, IdentityID: &id}
	if p.ToResponse().HasIdentity {
		t.Fatal("HasIdentity must be false when Identity association is not preloaded")
	}

	p.Identity = &Identity{}
	if p.ToResponse().HasIdentity {
		t.Fatal("HasIdentity must be false when AccessTokenEncrypted is nil")
	}

	empty := ""
	p.Identity.AccessTokenEncrypted = &empty
	if p.ToResponse().HasIdentity {
		t.Fatal("HasIdentity must be false when AccessTokenEncrypted is empty string")
	}

	tok := "encrypted-blob"
	p.Identity.AccessTokenEncrypted = &tok
	if !p.ToResponse().HasIdentity {
		t.Fatal("HasIdentity must be true once Identity has a non-empty AccessToken")
	}
}

func TestRepositoryProvider_ToResponse_HasBotTokenAndClientID(t *testing.T) {
	clientID := "id-123"
	botToken := "encrypted-token"
	p := &RepositoryProvider{
		ClientID:          &clientID,
		BotTokenEncrypted: &botToken,
	}
	resp := p.ToResponse()
	if !resp.HasClientID {
		t.Fatal("HasClientID must be true when ClientID set")
	}
	if !resp.HasBotToken {
		t.Fatal("HasBotToken must be true when BotTokenEncrypted set")
	}
}

func TestRepositoryProvider_ToResponse_JSONIncludesIsActiveField(t *testing.T) {
	p := &RepositoryProvider{ID: 1, ProviderType: ProviderTypeGitHub, Name: "GitHub", IsActive: true}
	body, err := json.Marshal(p.ToResponse())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["is_active"]; !ok {
		t.Fatalf("response JSON must contain `is_active` key, got keys: %v", keys(raw))
	}
	for _, k := range []string{"has_identity", "has_bot_token", "has_client_id"} {
		if _, ok := raw[k]; !ok {
			t.Fatalf("response JSON must contain %q key, got keys: %v", k, keys(raw))
		}
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
