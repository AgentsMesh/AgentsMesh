package extension

import (
	"encoding/json"
	"log/slog"
	"time"
)

// Scope constants
const (
	ScopeOrg  = "org"
	ScopeUser = "user"
)

// InstalledMcpServer represents an MCP Server installed for a specific repository
type InstalledMcpServer struct {
	ID             int64           `gorm:"primaryKey" json:"id"`
	OrganizationID int64           `gorm:"not null" json:"organization_id"`
	RepositoryID   int64           `gorm:"not null" json:"repository_id"`
	MarketItemID   *int64          `json:"market_item_id,omitempty"`
	Scope          string          `gorm:"size:20;not null" json:"scope"` // org / user
	InstalledBy    *int64          `json:"installed_by,omitempty"`
	Name           string          `gorm:"size:100" json:"name,omitempty"`
	Slug           string          `gorm:"size:100;not null" json:"slug"`
	TransportType  string          `gorm:"size:20;default:stdio" json:"transport_type"`
	Command        string          `gorm:"size:500" json:"command,omitempty"`
	Args           json.RawMessage `gorm:"type:jsonb;default:'[]'" json:"args,omitempty"`
	HttpURL        string          `gorm:"size:500" json:"http_url,omitempty"`
	HttpHeaders    json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"http_headers,omitempty"`
	EnvVars        json.RawMessage `gorm:"type:jsonb;default:'{}'" json:"env_vars,omitempty"` // encrypted
	IsEnabled      bool            `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt      time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null;default:now()" json:"updated_at"`

	// Relations
	MarketItem *McpMarketItem `gorm:"foreignKey:MarketItemID" json:"market_item,omitempty"`
}

func (InstalledMcpServer) TableName() string { return "installed_mcp_servers" }

// ToMcpConfig converts the installed server to an MCP config map entry.
// Merges with market item defaults if present.
func (s *InstalledMcpServer) ToMcpConfig() map[string]interface{} {
	config := make(map[string]interface{})

	if s.TransportType == TransportTypeHTTP || s.TransportType == TransportTypeSSE {
		config["type"] = s.TransportType
		url := s.HttpURL
		if url == "" && s.MarketItem != nil {
			url = s.MarketItem.DefaultHttpURL
		}
		config["url"] = url

		var headers map[string]string
		if len(s.HttpHeaders) > 0 {
			if err := json.Unmarshal(s.HttpHeaders, &headers); err != nil {
				slog.Warn("Failed to unmarshal MCP server http_headers",
					"server_id", s.ID, "slug", s.Slug, "error", err)
			}
		}
		if len(headers) > 0 {
			config["headers"] = headers
		}
	} else {
		// stdio transport
		command := s.Command
		if command == "" && s.MarketItem != nil {
			command = s.MarketItem.Command
		}
		config["command"] = command

		var args []string
		if len(s.Args) > 0 {
			if err := json.Unmarshal(s.Args, &args); err != nil {
				slog.Warn("Failed to unmarshal MCP server args",
					"server_id", s.ID, "slug", s.Slug, "error", err)
			}
		}
		if len(args) == 0 && s.MarketItem != nil {
			if err := json.Unmarshal(s.MarketItem.DefaultArgs, &args); err != nil {
				slog.Warn("Failed to unmarshal MCP market item default_args",
					"server_id", s.ID, "slug", s.Slug, "error", err)
			}
		}
		if len(args) > 0 {
			config["args"] = args
		}
	}

	// Merge env vars
	var envVars map[string]string
	if len(s.EnvVars) > 0 {
		if err := json.Unmarshal(s.EnvVars, &envVars); err != nil {
			slog.Warn("Failed to unmarshal MCP server env_vars",
				"server_id", s.ID, "slug", s.Slug, "error", err)
		}
	}
	if len(envVars) > 0 {
		config["env"] = envVars
	}

	return config
}
