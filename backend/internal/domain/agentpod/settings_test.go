package agentpod

import (
	"testing"
	"time"
)

// --- Test AI Provider Type Constants ---

func TestAIProviderTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{AIProviderTypeClaude, "claude"},
		{AIProviderTypeGemini, "gemini"},
		{AIProviderTypeCodex, "codex"},
		{AIProviderTypeOpenAI, "openai"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test UserAgentPodSettings ---

func TestUserAgentPodSettingsTableName(t *testing.T) {
	s := UserAgentPodSettings{}
	if s.TableName() != "user_agentpod_settings" {
		t.Errorf("expected 'user_agentpod_settings', got %s", s.TableName())
	}
}

func TestUserAgentPodSettingsStruct(t *testing.T) {
	now := time.Now()
	testSlug := "claude-code"
	model := "opus"
	permMode := "bypassPermissions"
	fontSize := 14
	theme := "dark"

	s := UserAgentPodSettings{
		ID: 1,
		UserID:             50,
		DefaultAgentSlug: &testSlug,
		DefaultModel:       &model,
		DefaultPermMode:    &permMode,
		TerminalFontSize:   &fontSize,
		TerminalTheme:      &theme,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if s.ID != 1 {
		t.Errorf("expected ID 1, got %d", s.ID)
	}
	if s.UserID != 50 {
		t.Errorf("expected UserID 50, got %d", s.UserID)
	}
	if *s.DefaultModel != "opus" {
		t.Errorf("expected DefaultModel 'opus', got %s", *s.DefaultModel)
	}
	if *s.TerminalFontSize != 14 {
		t.Errorf("expected TerminalFontSize 14, got %d", *s.TerminalFontSize)
	}
	if *s.TerminalTheme != "dark" {
		t.Errorf("expected TerminalTheme 'dark', got %s", *s.TerminalTheme)
	}
}

func TestUserAgentPodSettingsWithNilOptionalFields(t *testing.T) {
	s := UserAgentPodSettings{
		ID: 2,
		UserID: 50,
	}

	if s.DefaultAgentSlug != nil {
		t.Error("expected DefaultAgentSlug to be nil")
	}
	if s.DefaultModel != nil {
		t.Error("expected DefaultModel to be nil")
	}
	if s.DefaultPermMode != nil {
		t.Error("expected DefaultPermMode to be nil")
	}
	if s.TerminalFontSize != nil {
		t.Error("expected TerminalFontSize to be nil")
	}
	if s.TerminalTheme != nil {
		t.Error("expected TerminalTheme to be nil")
	}
}

// --- Test UserAIProvider ---

func TestUserAIProviderTableName(t *testing.T) {
	p := UserAIProvider{}
	if p.TableName() != "user_ai_providers" {
		t.Errorf("expected 'user_ai_providers', got %s", p.TableName())
	}
}

func TestUserAIProviderStruct(t *testing.T) {
	now := time.Now()

	p := UserAIProvider{
		ID: 1,
		UserID:               50,
		ProviderType:         AIProviderTypeClaude,
		Name:                 "My Claude",
		IsDefault:            true,
		IsEnabled:            true,
		EncryptedCredentials: "encrypted_data_here",
		LastUsedAt:           &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
	if p.UserID != 50 {
		t.Errorf("expected UserID 50, got %d", p.UserID)
	}
	if p.ProviderType != "claude" {
		t.Errorf("expected ProviderType 'claude', got %s", p.ProviderType)
	}
	if p.Name != "My Claude" {
		t.Errorf("expected Name 'My Claude', got %s", p.Name)
	}
	if !p.IsDefault {
		t.Error("expected IsDefault true")
	}
	if !p.IsEnabled {
		t.Error("expected IsEnabled true")
	}
	if p.EncryptedCredentials != "encrypted_data_here" {
		t.Errorf("expected EncryptedCredentials 'encrypted_data_here', got %s", p.EncryptedCredentials)
	}
}

func TestUserAIProviderWithAllProviderTypes(t *testing.T) {
	types := []string{AIProviderTypeClaude, AIProviderTypeGemini, AIProviderTypeCodex, AIProviderTypeOpenAI}

	for _, pt := range types {
		p := UserAIProvider{ProviderType: pt}
		if p.ProviderType != pt {
			t.Errorf("expected ProviderType '%s', got %s", pt, p.ProviderType)
		}
	}
}

// --- Test Credential Structs ---

func TestClaudeCredentialsStruct(t *testing.T) {
	creds := ClaudeCredentials{
		BaseURL:   "https://api.anthropic.com",
		AuthToken: "token123",
		APIKey:    "sk-xxx",
	}

	if creds.BaseURL != "https://api.anthropic.com" {
		t.Errorf("expected BaseURL 'https://api.anthropic.com', got %s", creds.BaseURL)
	}
	if creds.AuthToken != "token123" {
		t.Errorf("expected AuthToken 'token123', got %s", creds.AuthToken)
	}
	if creds.APIKey != "sk-xxx" {
		t.Errorf("expected APIKey 'sk-xxx', got %s", creds.APIKey)
	}
}

func TestOpenAICredentialsStruct(t *testing.T) {
	creds := OpenAICredentials{
		APIKey:       "sk-openai-xxx",
		Organization: "org-123",
		BaseURL:      "https://api.openai.com",
	}

	if creds.APIKey != "sk-openai-xxx" {
		t.Errorf("expected APIKey 'sk-openai-xxx', got %s", creds.APIKey)
	}
	if creds.Organization != "org-123" {
		t.Errorf("expected Organization 'org-123', got %s", creds.Organization)
	}
	if creds.BaseURL != "https://api.openai.com" {
		t.Errorf("expected BaseURL 'https://api.openai.com', got %s", creds.BaseURL)
	}
}

func TestGeminiCredentialsStruct(t *testing.T) {
	creds := GeminiCredentials{
		APIKey: "gemini-api-key",
	}

	if creds.APIKey != "gemini-api-key" {
		t.Errorf("expected APIKey 'gemini-api-key', got %s", creds.APIKey)
	}
}

// --- Test ProviderEnvVarMapping ---

func TestProviderEnvVarMappingClaude(t *testing.T) {
	mapping := ProviderEnvVarMapping[AIProviderTypeClaude]

	if mapping["base_url"] != "ANTHROPIC_BASE_URL" {
		t.Errorf("expected 'ANTHROPIC_BASE_URL', got %s", mapping["base_url"])
	}
	if mapping["auth_token"] != "ANTHROPIC_AUTH_TOKEN" {
		t.Errorf("expected 'ANTHROPIC_AUTH_TOKEN', got %s", mapping["auth_token"])
	}
	if mapping["api_key"] != "ANTHROPIC_API_KEY" {
		t.Errorf("expected 'ANTHROPIC_API_KEY', got %s", mapping["api_key"])
	}
}

func TestProviderEnvVarMappingOpenAI(t *testing.T) {
	mapping := ProviderEnvVarMapping[AIProviderTypeOpenAI]

	if mapping["api_key"] != "OPENAI_API_KEY" {
		t.Errorf("expected 'OPENAI_API_KEY', got %s", mapping["api_key"])
	}
	if mapping["organization"] != "OPENAI_ORG_ID" {
		t.Errorf("expected 'OPENAI_ORG_ID', got %s", mapping["organization"])
	}
	if mapping["base_url"] != "OPENAI_BASE_URL" {
		t.Errorf("expected 'OPENAI_BASE_URL', got %s", mapping["base_url"])
	}
}

func TestProviderEnvVarMappingGemini(t *testing.T) {
	mapping := ProviderEnvVarMapping[AIProviderTypeGemini]

	if mapping["api_key"] != "GOOGLE_API_KEY" {
		t.Errorf("expected 'GOOGLE_API_KEY', got %s", mapping["api_key"])
	}
}

func TestProviderEnvVarMappingCodex(t *testing.T) {
	mapping := ProviderEnvVarMapping[AIProviderTypeCodex]

	if mapping["api_key"] != "OPENAI_API_KEY" {
		t.Errorf("expected 'OPENAI_API_KEY', got %s", mapping["api_key"])
	}
	if mapping["organization"] != "OPENAI_ORG_ID" {
		t.Errorf("expected 'OPENAI_ORG_ID', got %s", mapping["organization"])
	}
}

func TestProviderEnvVarMappingAllProviders(t *testing.T) {
	providers := []string{AIProviderTypeClaude, AIProviderTypeOpenAI, AIProviderTypeGemini, AIProviderTypeCodex}

	for _, provider := range providers {
		mapping, exists := ProviderEnvVarMapping[provider]
		if !exists {
			t.Errorf("expected mapping to exist for provider '%s'", provider)
			continue
		}
		if len(mapping) == 0 {
			t.Errorf("expected non-empty mapping for provider '%s'", provider)
		}
	}
}

// --- Benchmark Tests ---

func BenchmarkUserAgentPodSettingsTableName(b *testing.B) {
	s := UserAgentPodSettings{}
	for i := 0; i < b.N; i++ {
		s.TableName()
	}
}

func BenchmarkUserAIProviderTableName(b *testing.B) {
	p := UserAIProvider{}
	for i := 0; i < b.N; i++ {
		p.TableName()
	}
}

func BenchmarkProviderEnvVarMappingLookup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ProviderEnvVarMapping[AIProviderTypeClaude]
	}
}
