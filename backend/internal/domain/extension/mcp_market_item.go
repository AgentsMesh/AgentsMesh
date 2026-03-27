package extension

import (
	"encoding/json"
	"time"
)

// Transport type constants
const (
	TransportTypeStdio = "stdio"
	TransportTypeHTTP  = "http"
	TransportTypeSSE   = "sse"
)

// MCP market item source constants
const (
	McpSourceSeed     = "seed"     // Built-in seed data from migrations
	McpSourceRegistry = "registry" // Synced from MCP Registry API
	McpSourceAdmin    = "admin"    // Manually added by admin
)

// McpMarketItem represents an MCP Server template in the marketplace
type McpMarketItem struct {
	ID                 int64           `gorm:"primaryKey" json:"id"`
	Slug               string          `gorm:"size:100;not null;uniqueIndex" json:"slug"`
	Name               string          `gorm:"size:100;not null" json:"name"`
	Description        string          `json:"description,omitempty"`
	Icon               string          `gorm:"size:50" json:"icon,omitempty"`
	TransportType      string          `gorm:"size:20;default:stdio" json:"transport_type"`
	Command            string          `gorm:"size:500" json:"command,omitempty"`
	DefaultArgs        json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"default_args,omitempty"`
	DefaultHttpURL     string          `gorm:"size:500" json:"default_http_url,omitempty"`
	DefaultHttpHeaders json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"default_http_headers,omitempty"`
	EnvVarSchema       json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"env_var_schema,omitempty"`
	AgentFilter    json.RawMessage `gorm:"type:jsonb" json:"agent_filter,omitempty"`
	Category           string          `gorm:"size:50" json:"category,omitempty"`
	IsActive           bool            `gorm:"not null;default:true" json:"is_active"`
	// Registry sync fields
	Source        string          `gorm:"size:20;default:seed" json:"source"`
	RegistryName  string          `gorm:"size:200" json:"registry_name,omitempty"`
	Version       string          `gorm:"size:50" json:"version,omitempty"`
	RepositoryURL string          `gorm:"size:500" json:"repository_url,omitempty"`
	RegistryMeta  json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"registry_meta,omitempty"`
	LastSyncedAt  *time.Time      `json:"last_synced_at,omitempty"`
	// Timestamps
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (McpMarketItem) TableName() string { return "mcp_market_items" }

// GetAgentFilter parses and returns the agent_filter as a string slice.
// Returns nil if the filter is empty or null (meaning all agents are allowed).
func (m *McpMarketItem) GetAgentFilter() []string {
	if len(m.AgentFilter) == 0 {
		return nil
	}
	var filter []string
	if err := json.Unmarshal(m.AgentFilter, &filter); err != nil {
		return nil
	}
	return filter
}

// EnvVarSchemaEntry represents a single env var definition in the schema
type EnvVarSchemaEntry struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Sensitive   bool   `json:"sensitive"`
	Placeholder string `json:"placeholder,omitempty"`
}
