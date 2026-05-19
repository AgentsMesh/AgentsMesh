package agentpod

import (
	"time"
)

type UserAgentPodSettings struct {
	ID     int64 `gorm:"primaryKey" json:"id"`
	UserID int64 `gorm:"not null;uniqueIndex" json:"user_id"`

	DefaultAgentSlug *string `gorm:"size:100;column:default_agent_slug" json:"default_agent_slug,omitempty"`
	DefaultModel       *string `gorm:"size:100" json:"default_model,omitempty"`
	DefaultPermMode    *string `gorm:"size:50" json:"default_perm_mode,omitempty"` // default, accept-edits, full-auto

	TerminalFontSize *int    `json:"terminal_font_size,omitempty"`
	TerminalTheme    *string `gorm:"size:50" json:"terminal_theme,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (UserAgentPodSettings) TableName() string {
	return "user_agentpod_settings"
}

type UserAIProvider struct {
	ID     int64 `gorm:"primaryKey" json:"id"`
	UserID int64 `gorm:"not null;index" json:"user_id"`

	ProviderType string `gorm:"size:50;not null;index" json:"provider_type"` // claude, gemini, codex, openai
	Name         string `gorm:"size:100;not null" json:"name"`               // User-defined name

	IsDefault bool `gorm:"not null;default:false" json:"is_default"` // Default for this provider type
	IsEnabled bool `gorm:"not null;default:true" json:"is_enabled"`

	EncryptedCredentials string `gorm:"type:text;not null" json:"-"`

	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (UserAIProvider) TableName() string {
	return "user_ai_providers"
}

const (
	AIProviderTypeClaude = "claude"
	AIProviderTypeGemini = "gemini"
	AIProviderTypeCodex  = "codex"
	AIProviderTypeOpenAI = "openai"
)

type ClaudeCredentials struct {
	BaseURL   string `json:"base_url,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
	APIKey    string `json:"api_key,omitempty"` // Alternative to auth_token
}

type OpenAICredentials struct {
	APIKey       string `json:"api_key"`
	Organization string `json:"organization,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

type GeminiCredentials struct {
	APIKey string `json:"api_key"`
}

var ProviderEnvVarMapping = map[string]map[string]string{
	AIProviderTypeClaude: {
		"base_url":   "ANTHROPIC_BASE_URL",
		"auth_token": "ANTHROPIC_AUTH_TOKEN",
		"api_key":    "ANTHROPIC_API_KEY",
	},
	AIProviderTypeOpenAI: {
		"api_key":      "OPENAI_API_KEY",
		"organization": "OPENAI_ORG_ID",
		"base_url":     "OPENAI_BASE_URL",
	},
	AIProviderTypeGemini: {
		"api_key": "GOOGLE_API_KEY",
	},
	AIProviderTypeCodex: {
		"api_key":      "OPENAI_API_KEY",
		"organization": "OPENAI_ORG_ID",
	},
}
