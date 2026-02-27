package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// McpRegistryClient communicates with the official MCP Registry API
// (https://registry.modelcontextprotocol.io).
type McpRegistryClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewMcpRegistryClient creates a client for the MCP Registry API.
func NewMcpRegistryClient(baseURL string) *McpRegistryClient {
	return &McpRegistryClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// --- API response types ---

// RegistryResponse is the top-level response from GET /v0/servers.
type RegistryResponse struct {
	Servers  []RegistryServerEntry `json:"servers"`
	Metadata RegistryMetadata      `json:"metadata"`
}

// RegistryServerEntry wraps a single server + registry metadata.
type RegistryServerEntry struct {
	Server RegistryServer  `json:"server"`
	Meta   json.RawMessage `json:"_meta"`
}

// RegistryServer is the server.json payload.
type RegistryServer struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Title       string              `json:"title"`
	Version     string              `json:"version"`
	WebsiteURL  string              `json:"websiteUrl"`
	Repository  *RegistryRepository `json:"repository"`
	Packages    []RegistryPackage   `json:"packages"`
	Remotes     []RegistryRemote    `json:"remotes"`
}

// RegistryRepository holds the source repository info.
type RegistryRepository struct {
	URL       string `json:"url"`
	Source    string `json:"source"`
	Subfolder string `json:"subfolder"`
}

// RegistryPackage describes a local package (npm/pypi/oci).
type RegistryPackage struct {
	RegistryType         string             `json:"registryType"`
	Identifier           string             `json:"identifier"`
	Version              string             `json:"version"`
	Transport            RegistryTransport  `json:"transport"`
	EnvironmentVariables []RegistryEnvVar   `json:"environmentVariables"`
}

// RegistryRemote describes a remote server endpoint (sse/http).
type RegistryRemote struct {
	Type    string           `json:"type"`
	URL     string           `json:"url"`
	Headers []RegistryHeader `json:"headers"`
}

// RegistryTransport holds transport type info.
type RegistryTransport struct {
	Type string `json:"type"`
}

// RegistryEnvVar is an environment variable definition from the registry.
type RegistryEnvVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsRequired  bool   `json:"isRequired"`
	IsSecret    bool   `json:"isSecret"`
	Default     string `json:"default,omitempty"`
	Format      string `json:"format,omitempty"`
}

// RegistryHeader is an HTTP header definition for remote servers.
type RegistryHeader struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Value       string `json:"value,omitempty"`
	IsRequired  bool   `json:"isRequired"`
	IsSecret    bool   `json:"isSecret"`
}

// RegistryMetadata holds pagination info.
type RegistryMetadata struct {
	NextCursor string `json:"nextCursor"`
	Count      int    `json:"count"`
}

// RegistryOfficialMeta is the parsed _meta.io.modelcontextprotocol.registry/official.
type RegistryOfficialMeta struct {
	Status      string `json:"status"`
	PublishedAt string `json:"publishedAt"`
	UpdatedAt   string `json:"updatedAt"`
	IsLatest    bool   `json:"isLatest"`
}

// maxRegistryPages is the upper bound on pages we will fetch from the registry
// to prevent infinite pagination loops.
const maxRegistryPages = 500

// --- Public methods ---

// FetchAll pages through the registry and returns all active, latest server entries.
func (c *McpRegistryClient) FetchAll(ctx context.Context) ([]RegistryServerEntry, error) {
	var all []RegistryServerEntry
	cursor := ""
	pageNum := 0

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		page, err := c.FetchPage(ctx, cursor, 100)
		if err != nil {
			return nil, fmt.Errorf("fetch page %d: %w", pageNum, err)
		}
		pageNum++

		for _, entry := range page.Servers {
			// Only keep entries marked as latest and active
			if !c.isLatestActive(entry.Meta) {
				continue
			}
			all = append(all, entry)
		}

		slog.Debug("MCP Registry: fetched page",
			"page", pageNum, "count", len(page.Servers), "total_kept", len(all))

		if page.Metadata.NextCursor == "" || len(page.Servers) == 0 {
			break
		}
		cursor = page.Metadata.NextCursor

		if pageNum >= maxRegistryPages {
			slog.Warn("MCP Registry: reached max page limit, stopping pagination",
				"maxPages", maxRegistryPages, "total_kept", len(all))
			break
		}
	}

	return all, nil
}

// FetchPage fetches a single page of servers from the registry.
func (c *McpRegistryClient) FetchPage(ctx context.Context, cursor string, limit int) (*RegistryResponse, error) {
	u, err := url.Parse(c.baseURL + "/v0/servers")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	q := u.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "AgentsMesh-Backend/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("registry returned %d: %s", resp.StatusCode, string(body))
	}

	var result RegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// isLatestActive checks if the entry is the latest version and active.
func (c *McpRegistryClient) isLatestActive(meta json.RawMessage) bool {
	if len(meta) == 0 {
		return true // if no meta, assume latest
	}
	// Parse the nested _meta structure
	var metaMap map[string]json.RawMessage
	if err := json.Unmarshal(meta, &metaMap); err != nil {
		return true
	}
	officialRaw, ok := metaMap["io.modelcontextprotocol.registry/official"]
	if !ok {
		return true
	}
	var official RegistryOfficialMeta
	if err := json.Unmarshal(officialRaw, &official); err != nil {
		return true
	}
	return official.IsLatest && official.Status == "active"
}
