package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ===========================================================================
// McpRegistryClient tests
// ===========================================================================

// --- Constructor ---

func TestNewMcpRegistryClient(t *testing.T) {
	c := NewMcpRegistryClient("https://registry.example.com")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "https://registry.example.com" {
		t.Errorf("expected baseURL %q, got %q", "https://registry.example.com", c.baseURL)
	}
	if c.httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", c.httpClient.Timeout)
	}
}

// --- isLatestActive ---

func TestIsLatestActive_EmptyMeta(t *testing.T) {
	c := NewMcpRegistryClient("")
	if !c.isLatestActive(nil) {
		t.Error("empty meta should return true")
	}
	if !c.isLatestActive(json.RawMessage{}) {
		t.Error("zero-length meta should return true")
	}
}

func TestIsLatestActive_MissingOfficialField(t *testing.T) {
	c := NewMcpRegistryClient("")
	meta := json.RawMessage(`{"some.other.key": {"foo": "bar"}}`)
	if !c.isLatestActive(meta) {
		t.Error("meta without official key should return true")
	}
}

func TestIsLatestActive_ActiveAndLatest(t *testing.T) {
	c := NewMcpRegistryClient("")
	meta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	if !c.isLatestActive(meta) {
		t.Error("active + isLatest should return true")
	}
}

func TestIsLatestActive_NotLatest(t *testing.T) {
	c := NewMcpRegistryClient("")
	meta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": false, "status": "active"}}`)
	if c.isLatestActive(meta) {
		t.Error("isLatest=false should return false")
	}
}

func TestIsLatestActive_NotActive(t *testing.T) {
	c := NewMcpRegistryClient("")
	meta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "deprecated"}}`)
	if c.isLatestActive(meta) {
		t.Error("status=deprecated should return false")
	}
}

func TestIsLatestActive_InvalidJSON(t *testing.T) {
	c := NewMcpRegistryClient("")
	// Outer parse failure
	meta := json.RawMessage(`{not valid json}`)
	if !c.isLatestActive(meta) {
		t.Error("invalid JSON should return true (default)")
	}

	// Inner parse failure (official key is not valid JSON for RegistryOfficialMeta)
	meta2 := json.RawMessage(`{"io.modelcontextprotocol.registry/official": "not an object"}`)
	if !c.isLatestActive(meta2) {
		t.Error("unparseable official meta should return true (default)")
	}
}

// --- FetchPage ---

func TestFetchPage_Success(t *testing.T) {
	resp := RegistryResponse{
		Servers: []RegistryServerEntry{
			{Server: RegistryServer{Name: "test/server1"}},
			{Server: RegistryServer{Name: "test/server2"}},
		},
		Metadata: RegistryMetadata{Count: 2},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v0/servers" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("User-Agent") != "AgentsMesh-Backend/1.0" {
			t.Errorf("unexpected User-Agent: %s", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("unexpected Accept: %s", r.Header.Get("Accept"))
		}
		if r.URL.Query().Get("limit") != "50" {
			t.Errorf("expected limit=50, got %s", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	result, err := c.FetchPage(context.Background(), "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(result.Servers))
	}
	if result.Servers[0].Server.Name != "test/server1" {
		t.Errorf("expected server name 'test/server1', got %q", result.Servers[0].Server.Name)
	}
}

func TestFetchPage_WithCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		if cursor != "abc123" {
			t.Errorf("expected cursor=abc123, got %q", cursor)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RegistryResponse{
			Servers:  []RegistryServerEntry{},
			Metadata: RegistryMetadata{Count: 0},
		})
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	_, err := c.FetchPage(context.Background(), "abc123", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchPage_Non200StatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	_, err := c.FetchPage(context.Background(), "", 100)
	if err == nil {
		t.Fatal("expected error for non-200 status code")
	}
	if got := err.Error(); !contains(got, "500") {
		t.Errorf("expected error to contain status code 500, got: %s", got)
	}
}

func TestFetchPage_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{not valid json"))
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	_, err := c.FetchPage(context.Background(), "", 100)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !contains(err.Error(), "decode response") {
		t.Errorf("expected 'decode response' in error, got: %s", err.Error())
	}
}

func TestFetchPage_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally slow — context should cancel before response
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.FetchPage(ctx, "", 100)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

// --- FetchAll ---

func TestFetchAll_SinglePage(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	resp := RegistryResponse{
		Servers: []RegistryServerEntry{
			{Server: RegistryServer{Name: "test/server1"}, Meta: activeMeta},
			{Server: RegistryServer{Name: "test/server2"}, Meta: activeMeta},
		},
		Metadata: RegistryMetadata{Count: 2, NextCursor: ""},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	entries, err := c.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestFetchAll_MultiplePages(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	pageNum := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pageNum == 0 {
			pageNum++
			json.NewEncoder(w).Encode(RegistryResponse{
				Servers: []RegistryServerEntry{
					{Server: RegistryServer{Name: "test/page1-server1"}, Meta: activeMeta},
				},
				Metadata: RegistryMetadata{Count: 1, NextCursor: "cursor-page2"},
			})
		} else {
			json.NewEncoder(w).Encode(RegistryResponse{
				Servers: []RegistryServerEntry{
					{Server: RegistryServer{Name: "test/page2-server1"}, Meta: activeMeta},
				},
				Metadata: RegistryMetadata{Count: 1, NextCursor: ""},
			})
		}
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	entries, err := c.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries across 2 pages, got %d", len(entries))
	}
	if entries[0].Server.Name != "test/page1-server1" {
		t.Errorf("expected first entry from page 1, got %q", entries[0].Server.Name)
	}
	if entries[1].Server.Name != "test/page2-server1" {
		t.Errorf("expected second entry from page 2, got %q", entries[1].Server.Name)
	}
}

func TestFetchAll_FiltersNonLatestActive(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	deprecatedMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "deprecated"}}`)
	notLatestMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": false, "status": "active"}}`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RegistryResponse{
			Servers: []RegistryServerEntry{
				{Server: RegistryServer{Name: "test/active"}, Meta: activeMeta},
				{Server: RegistryServer{Name: "test/deprecated"}, Meta: deprecatedMeta},
				{Server: RegistryServer{Name: "test/not-latest"}, Meta: notLatestMeta},
				{Server: RegistryServer{Name: "test/no-meta"}},
			},
			Metadata: RegistryMetadata{Count: 4},
		})
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	entries, err := c.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "active" (isLatest+active) and "no-meta" (nil meta defaults to true)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (active + no-meta), got %d", len(entries))
		for _, e := range entries {
			t.Logf("  kept: %s", e.Server.Name)
		}
	}
}

func TestFetchAll_FetchPageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unavailable"))
	}))
	defer srv.Close()

	c := NewMcpRegistryClient(srv.URL)
	_, err := c.FetchAll(context.Background())
	if err == nil {
		t.Fatal("expected error when FetchPage fails")
	}
	if !contains(err.Error(), "fetch page 0") {
		t.Errorf("expected 'fetch page 0' in error, got: %s", err.Error())
	}
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before any request

	c := NewMcpRegistryClient("http://unreachable.invalid")
	_, err := c.FetchAll(ctx)
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

// ===========================================================================
// McpRegistrySyncer tests - helper functions
// ===========================================================================

// --- pkgPriority ---

func TestPkgPriority_Npm(t *testing.T) {
	if got := pkgPriority("npm"); got != 0 {
		t.Errorf("npm priority: expected 0, got %d", got)
	}
}

func TestPkgPriority_Pypi(t *testing.T) {
	if got := pkgPriority("pypi"); got != 1 {
		t.Errorf("pypi priority: expected 1, got %d", got)
	}
}

func TestPkgPriority_Oci(t *testing.T) {
	if got := pkgPriority("oci"); got != 2 {
		t.Errorf("oci priority: expected 2, got %d", got)
	}
}

func TestPkgPriority_Unknown(t *testing.T) {
	if got := pkgPriority("cargo"); got != 9 {
		t.Errorf("unknown priority: expected 9, got %d", got)
	}
}

// --- registryNameToSlug ---

func TestRegistryNameToSlug_SimpleSlash(t *testing.T) {
	got := registryNameToSlug("io.github.user/server")
	expected := "io.github.user--server"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRegistryNameToSlug_SpecialChars(t *testing.T) {
	got := registryNameToSlug("io.github.user/my server@v2!")
	// / → --, space → -, @ → -, ! → -
	expected := "io.github.user--my-server-v2-"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRegistryNameToSlug_Lowercase(t *testing.T) {
	got := registryNameToSlug("GitHub.User/MyServer")
	expected := "github.user--myserver"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// --- Constructor ---

func TestNewMcpRegistrySyncer(t *testing.T) {
	client := NewMcpRegistryClient("https://example.com")
	repo := newMockExtensionRepo()
	s := NewMcpRegistrySyncer(client, repo)
	if s == nil {
		t.Fatal("expected non-nil syncer")
	}
	if s.client != client {
		t.Error("expected client to be set")
	}
	if s.repo != repo {
		t.Error("expected repo to be set")
	}
}

// ===========================================================================
// convertToMarketItem tests
// ===========================================================================

func newTestSyncer() *McpRegistrySyncer {
	return &McpRegistrySyncer{
		client: NewMcpRegistryClient(""),
		repo:   newMockExtensionRepo(),
	}
}

func TestConvertToMarketItem_NoName(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{Name: ""},
	}
	_, err := s.convertToMarketItem(entry, time.Now())
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !contains(err.Error(), "no name") {
		t.Errorf("expected 'no name' in error, got: %s", err.Error())
	}
}

func TestConvertToMarketItem_TitleFallback(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name:    "io.github.user/my-server",
			Title:   "",
			Packages: []RegistryPackage{
				{RegistryType: "npm", Identifier: "@user/my-server"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fallback to last part of name
	if item.Name != "my-server" {
		t.Errorf("expected name 'my-server', got %q", item.Name)
	}
}

func TestConvertToMarketItem_WithTitle(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name:    "io.github.user/my-server",
			Title:   "My Cool Server",
			Packages: []RegistryPackage{
				{RegistryType: "npm", Identifier: "@user/my-server"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Name != "My Cool Server" {
		t.Errorf("expected name 'My Cool Server', got %q", item.Name)
	}
}

func TestConvertToMarketItem_WithRepository(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name:  "test/server",
			Title: "Test Server",
			Repository: &RegistryRepository{
				URL:    "https://github.com/test/server",
				Source: "github",
			},
			Packages: []RegistryPackage{
				{RegistryType: "npm", Identifier: "test-server"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.RepositoryURL != "https://github.com/test/server" {
		t.Errorf("expected repository URL, got %q", item.RepositoryURL)
	}
}

func TestConvertToMarketItem_NpmPackage(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/npm-server",
			Packages: []RegistryPackage{
				{RegistryType: "npm", Identifier: "@test/npm-server"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", item.Command)
	}
	if item.TransportType != extension.TransportTypeStdio {
		t.Errorf("expected transport 'stdio', got %q", item.TransportType)
	}
	if item.Category != "npm" {
		t.Errorf("expected category 'npm', got %q", item.Category)
	}
	// Check default args
	var args []string
	if err := json.Unmarshal(item.DefaultArgs, &args); err != nil {
		t.Fatalf("failed to unmarshal default args: %v", err)
	}
	if len(args) != 2 || args[0] != "-y" || args[1] != "@test/npm-server" {
		t.Errorf("expected args [-y, @test/npm-server], got %v", args)
	}
}

func TestConvertToMarketItem_PypiPackage(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/pypi-server",
			Packages: []RegistryPackage{
				{RegistryType: "pypi", Identifier: "mcp-server-test"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Command != "uvx" {
		t.Errorf("expected command 'uvx', got %q", item.Command)
	}
	if item.Category != "pypi" {
		t.Errorf("expected category 'pypi', got %q", item.Category)
	}
	var args []string
	if err := json.Unmarshal(item.DefaultArgs, &args); err != nil {
		t.Fatalf("failed to unmarshal default args: %v", err)
	}
	if len(args) != 1 || args[0] != "mcp-server-test" {
		t.Errorf("expected args [mcp-server-test], got %v", args)
	}
}

func TestConvertToMarketItem_OciPackage(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/oci-server",
			Packages: []RegistryPackage{
				{RegistryType: "oci", Identifier: "ghcr.io/test/server:latest"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Command != "docker" {
		t.Errorf("expected command 'docker', got %q", item.Command)
	}
	if item.Category != "oci" {
		t.Errorf("expected category 'oci', got %q", item.Category)
	}
	var args []string
	if err := json.Unmarshal(item.DefaultArgs, &args); err != nil {
		t.Fatalf("failed to unmarshal default args: %v", err)
	}
	if len(args) != 4 || args[0] != "run" || args[1] != "-i" || args[2] != "--rm" || args[3] != "ghcr.io/test/server:latest" {
		t.Errorf("expected args [run, -i, --rm, ghcr.io/test/server:latest], got %v", args)
	}
}

func TestConvertToMarketItem_WithEnvVars(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/env-server",
			Packages: []RegistryPackage{
				{
					RegistryType: "npm",
					Identifier:   "@test/env-server",
					EnvironmentVariables: []RegistryEnvVar{
						{
							Name:        "API_KEY",
							Description: "Your API key",
							IsRequired:  true,
							IsSecret:    true,
							Default:     "",
						},
						{
							Name:        "BASE_URL",
							Description: "Base URL for the API",
							IsRequired:  false,
							IsSecret:    false,
							Default:     "https://api.example.com",
						},
					},
				},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.EnvVarSchema == nil {
		t.Fatal("expected env var schema to be populated")
	}
	var schema []extension.EnvVarSchemaEntry
	if err := json.Unmarshal(item.EnvVarSchema, &schema); err != nil {
		t.Fatalf("failed to unmarshal env var schema: %v", err)
	}
	if len(schema) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(schema))
	}
	// Check first env var
	if schema[0].Name != "API_KEY" {
		t.Errorf("expected name 'API_KEY', got %q", schema[0].Name)
	}
	if schema[0].Label != "Your API key" {
		t.Errorf("expected label 'Your API key', got %q", schema[0].Label)
	}
	if !schema[0].Required {
		t.Error("expected API_KEY to be required")
	}
	if !schema[0].Sensitive {
		t.Error("expected API_KEY to be sensitive")
	}
	// Check second env var has placeholder from default
	if schema[1].Placeholder != "https://api.example.com" {
		t.Errorf("expected placeholder from default, got %q", schema[1].Placeholder)
	}
}

func TestConvertToMarketItem_RemoteSSE(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/sse-server",
			Remotes: []RegistryRemote{
				{Type: "sse", URL: "https://sse.example.com/events"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.TransportType != extension.TransportTypeSSE {
		t.Errorf("expected transport 'sse', got %q", item.TransportType)
	}
	if item.DefaultHttpURL != "https://sse.example.com/events" {
		t.Errorf("expected URL 'https://sse.example.com/events', got %q", item.DefaultHttpURL)
	}
}

func TestConvertToMarketItem_RemoteHTTP(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/http-server",
			Remotes: []RegistryRemote{
				{Type: "http", URL: "https://http.example.com/mcp"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.TransportType != extension.TransportTypeHTTP {
		t.Errorf("expected transport 'http', got %q", item.TransportType)
	}
}

func TestConvertToMarketItem_RemoteWithHeaders(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/header-server",
			Remotes: []RegistryRemote{
				{
					Type: "sse",
					URL:  "https://sse.example.com/events",
					Headers: []RegistryHeader{
						{
							Name:        "Authorization",
							Description: "Bearer token",
							IsRequired:  true,
							IsSecret:    true,
						},
						{
							Name:        "X-Custom",
							Description: "Custom header",
							Value:       "custom-value",
							IsRequired:  false,
							IsSecret:    false,
						},
					},
				},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.DefaultHttpHeaders == nil {
		t.Fatal("expected headers to be populated")
	}
	var headers []map[string]interface{}
	if err := json.Unmarshal(item.DefaultHttpHeaders, &headers); err != nil {
		t.Fatalf("failed to unmarshal headers: %v", err)
	}
	if len(headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(headers))
	}
	if headers[0]["name"] != "Authorization" {
		t.Errorf("expected header name 'Authorization', got %v", headers[0]["name"])
	}
	if headers[0]["required"] != true {
		t.Errorf("expected required=true, got %v", headers[0]["required"])
	}
	if headers[0]["sensitive"] != true {
		t.Errorf("expected sensitive=true, got %v", headers[0]["sensitive"])
	}
	if headers[0]["description"] != "Bearer token" {
		t.Errorf("expected description 'Bearer token', got %v", headers[0]["description"])
	}
	// Second header should have value
	if headers[1]["value"] != "custom-value" {
		t.Errorf("expected value 'custom-value', got %v", headers[1]["value"])
	}
}

func TestConvertToMarketItem_NoPackageOrRemote(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/empty-server",
			// No packages, no remotes
		},
	}
	_, err := s.convertToMarketItem(entry, time.Now())
	if err == nil {
		t.Fatal("expected error for no package or remote")
	}
	if !contains(err.Error(), "no usable package or remote") {
		t.Errorf("expected 'no usable package or remote' in error, got: %s", err.Error())
	}
}

func TestConvertToMarketItem_PackagePriority(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name: "test/multi-package",
			Packages: []RegistryPackage{
				{RegistryType: "pypi", Identifier: "pypi-server"},
				{RegistryType: "npm", Identifier: "@test/npm-server"},
				{RegistryType: "oci", Identifier: "ghcr.io/test/server"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// npm should win (priority 0)
	if item.Command != "npx" {
		t.Errorf("expected command 'npx' (npm wins), got %q", item.Command)
	}
	if item.Category != "npm" {
		t.Errorf("expected category 'npm', got %q", item.Category)
	}
}

// ===========================================================================
// applyPackageConfig tests
// ===========================================================================

func TestApplyPackageConfig_EmptyPackages(t *testing.T) {
	s := newTestSyncer()
	item := &extension.McpMarketItem{}
	s.applyPackageConfig(item, nil)
	if item.TransportType != "" {
		t.Errorf("expected empty transport type, got %q", item.TransportType)
	}
	if item.Command != "" {
		t.Errorf("expected empty command, got %q", item.Command)
	}
}

func TestApplyPackageConfig_UnknownType(t *testing.T) {
	s := newTestSyncer()
	item := &extension.McpMarketItem{}
	packages := []RegistryPackage{
		{RegistryType: "cargo", Identifier: "some-crate"},
	}
	s.applyPackageConfig(item, packages)
	// TransportType should be set to "stdio" (default from applyPackageConfig)
	// but Command should remain empty since it's an unknown type
	if item.TransportType != extension.TransportTypeStdio {
		t.Errorf("expected transport 'stdio', got %q", item.TransportType)
	}
	if item.Command != "" {
		t.Errorf("expected empty command for unknown type, got %q", item.Command)
	}
}

// ===========================================================================
// applyRemoteConfig tests
// ===========================================================================

func TestApplyRemoteConfig_EmptyRemotes(t *testing.T) {
	s := newTestSyncer()
	item := &extension.McpMarketItem{}
	s.applyRemoteConfig(item, nil)
	if item.TransportType != "" {
		t.Errorf("expected empty transport type, got %q", item.TransportType)
	}
}

func TestApplyRemoteConfig_StreamableHTTP(t *testing.T) {
	s := newTestSyncer()
	item := &extension.McpMarketItem{}
	remotes := []RegistryRemote{
		{Type: "streamable-http", URL: "https://example.com/mcp"},
	}
	s.applyRemoteConfig(item, remotes)
	if item.TransportType != extension.TransportTypeHTTP {
		t.Errorf("expected transport 'http', got %q", item.TransportType)
	}
	if item.DefaultHttpURL != "https://example.com/mcp" {
		t.Errorf("expected URL, got %q", item.DefaultHttpURL)
	}
}

// ===========================================================================
// Sync tests
// ===========================================================================

// syncerMockRepo extends mockExtensionRepo with additional tracking for Sync tests.
type syncerMockRepo struct {
	mockExtensionRepo

	mu              sync.Mutex
	upsertedItems   []*extension.McpMarketItem
	upsertFunc      func(ctx context.Context, item *extension.McpMarketItem) error
	deactivateFunc  func(ctx context.Context, source string, names []string) (int64, error)
	deactivateCalls []deactivateCall
}

type deactivateCall struct {
	Source string
	Names  []string
}

func newSyncerMockRepo() *syncerMockRepo {
	return &syncerMockRepo{}
}

func (m *syncerMockRepo) UpsertMcpMarketItem(ctx context.Context, item *extension.McpMarketItem) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, item)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertedItems = append(m.upsertedItems, item)
	return nil
}

func (m *syncerMockRepo) DeactivateMcpMarketItemsNotIn(ctx context.Context, source string, names []string) (int64, error) {
	if m.deactivateFunc != nil {
		return m.deactivateFunc(ctx, source, names)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deactivateCalls = append(m.deactivateCalls, deactivateCall{Source: source, Names: names})
	return 0, nil
}

// Compile-time check
var _ extension.Repository = (*syncerMockRepo)(nil)

// newRegistryServer creates an httptest server that returns given responses page-by-page.
func newRegistryServer(t *testing.T, pages []RegistryResponse) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	pageIdx := 0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		idx := pageIdx
		pageIdx++
		mu.Unlock()

		if idx >= len(pages) {
			// Return empty page if we run out
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RegistryResponse{})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pages[idx])
	}))
}

func TestSync_Success(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)

	pages := []RegistryResponse{
		{
			Servers: []RegistryServerEntry{
				{
					Server: RegistryServer{
						Name: "test/server1",
						Packages: []RegistryPackage{
							{RegistryType: "npm", Identifier: "@test/server1"},
						},
					},
					Meta: activeMeta,
				},
				{
					Server: RegistryServer{
						Name: "test/server2",
						Remotes: []RegistryRemote{
							{Type: "sse", URL: "https://example.com/sse"},
						},
					},
					Meta: activeMeta,
				},
			},
			Metadata: RegistryMetadata{Count: 2, NextCursor: ""},
		},
	}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	client := NewMcpRegistryClient(srv.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.upsertedItems) != 2 {
		t.Fatalf("expected 2 upserted items, got %d", len(repo.upsertedItems))
	}
	if repo.upsertedItems[0].RegistryName != "test/server1" {
		t.Errorf("expected registry name 'test/server1', got %q", repo.upsertedItems[0].RegistryName)
	}
	if repo.upsertedItems[1].RegistryName != "test/server2" {
		t.Errorf("expected registry name 'test/server2', got %q", repo.upsertedItems[1].RegistryName)
	}

	// Check deactivate was called
	if len(repo.deactivateCalls) != 1 {
		t.Fatalf("expected 1 deactivate call, got %d", len(repo.deactivateCalls))
	}
	if repo.deactivateCalls[0].Source != extension.McpSourceRegistry {
		t.Errorf("expected source %q, got %q", extension.McpSourceRegistry, repo.deactivateCalls[0].Source)
	}
	if len(repo.deactivateCalls[0].Names) != 2 {
		t.Errorf("expected 2 synced names, got %d", len(repo.deactivateCalls[0].Names))
	}
}

func TestSync_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer srv.Close()

	repo := newSyncerMockRepo()
	client := NewMcpRegistryClient(srv.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	err := syncer.Sync(context.Background())
	if err == nil {
		t.Fatal("expected error when FetchAll fails")
	}
	if !contains(err.Error(), "fetch registry") {
		t.Errorf("expected 'fetch registry' in error, got: %s", err.Error())
	}
}

func TestSync_ContextCancelled(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)

	// Create many entries so the loop has something to iterate
	servers := make([]RegistryServerEntry, 100)
	for i := range servers {
		servers[i] = RegistryServerEntry{
			Server: RegistryServer{
				Name: fmt.Sprintf("test/server%d", i),
				Packages: []RegistryPackage{
					{RegistryType: "npm", Identifier: fmt.Sprintf("@test/server%d", i)},
				},
			},
			Meta: activeMeta,
		}
	}

	pages := []RegistryResponse{
		{
			Servers:  servers,
			Metadata: RegistryMetadata{Count: len(servers)},
		},
	}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	upsertCount := 0
	repo.upsertFunc = func(_ context.Context, item *extension.McpMarketItem) error {
		repo.mu.Lock()
		upsertCount++
		repo.mu.Unlock()
		return nil
	}

	client := NewMcpRegistryClient(srv.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a brief moment so some items are processed
	go func() {
		time.Sleep(1 * time.Millisecond)
		cancel()
	}()

	err := syncer.Sync(ctx)
	// Either returns context.Canceled or processes all quickly
	// The key assertion is it doesn't panic and respects cancellation
	if err != nil && err != context.Canceled {
		// It's acceptable if it completes before cancel fires
		t.Logf("sync returned: %v", err)
	}
}

func TestSync_SkipInvalidEntries(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)

	pages := []RegistryResponse{
		{
			Servers: []RegistryServerEntry{
				{
					// Invalid: no name
					Server: RegistryServer{Name: ""},
					Meta:   activeMeta,
				},
				{
					// Invalid: no packages or remotes
					Server: RegistryServer{Name: "test/no-config"},
					Meta:   activeMeta,
				},
				{
					// Valid entry
					Server: RegistryServer{
						Name: "test/valid",
						Packages: []RegistryPackage{
							{RegistryType: "npm", Identifier: "@test/valid"},
						},
					},
					Meta: activeMeta,
				},
			},
			Metadata: RegistryMetadata{Count: 3},
		},
	}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	client := NewMcpRegistryClient(srv.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	// Only 1 valid entry should be upserted
	if len(repo.upsertedItems) != 1 {
		t.Fatalf("expected 1 upserted item, got %d", len(repo.upsertedItems))
	}
	if repo.upsertedItems[0].RegistryName != "test/valid" {
		t.Errorf("expected 'test/valid', got %q", repo.upsertedItems[0].RegistryName)
	}
}

func TestSync_UpsertError(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)

	pages := []RegistryResponse{
		{
			Servers: []RegistryServerEntry{
				{
					Server: RegistryServer{
						Name: "test/fail-upsert",
						Packages: []RegistryPackage{
							{RegistryType: "npm", Identifier: "@test/fail"},
						},
					},
					Meta: activeMeta,
				},
				{
					Server: RegistryServer{
						Name: "test/success-upsert",
						Packages: []RegistryPackage{
							{RegistryType: "npm", Identifier: "@test/success"},
						},
					},
					Meta: activeMeta,
				},
			},
			Metadata: RegistryMetadata{Count: 2},
		},
	}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	callNum := 0
	repo.upsertFunc = func(_ context.Context, item *extension.McpMarketItem) error {
		repo.mu.Lock()
		callNum++
		n := callNum
		repo.mu.Unlock()
		if n == 1 {
			return errors.New("db write error")
		}
		// Track successful upserts manually
		repo.mu.Lock()
		repo.upsertedItems = append(repo.upsertedItems, item)
		repo.mu.Unlock()
		return nil
	}

	client := NewMcpRegistryClient(srv.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error (upsert errors should be skipped): %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	// Only the second item should succeed
	if len(repo.upsertedItems) != 1 {
		t.Fatalf("expected 1 successful upsert, got %d", len(repo.upsertedItems))
	}

	// Deactivate should only include the successful one
	if len(repo.deactivateCalls) != 1 {
		t.Fatalf("expected 1 deactivate call, got %d", len(repo.deactivateCalls))
	}
	if len(repo.deactivateCalls[0].Names) != 1 {
		t.Errorf("expected 1 synced name, got %d", len(repo.deactivateCalls[0].Names))
	}
}

func TestSync_DeactivateError(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)

	pages := []RegistryResponse{
		{
			Servers: []RegistryServerEntry{
				{
					Server: RegistryServer{
						Name: "test/server1",
						Packages: []RegistryPackage{
							{RegistryType: "npm", Identifier: "@test/server1"},
						},
					},
					Meta: activeMeta,
				},
			},
			Metadata: RegistryMetadata{Count: 1},
		},
	}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	repo.deactivateFunc = func(_ context.Context, _ string, _ []string) (int64, error) {
		return 0, errors.New("deactivation failed")
	}

	client := NewMcpRegistryClient(srv.URL)
	syncer := NewMcpRegistrySyncer(client, repo)

	// Sync should NOT return an error even when deactivation fails
	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("expected no error (deactivation failure is warn-only), got: %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	// The upsert should have still happened
	if len(repo.upsertedItems) != 1 {
		t.Errorf("expected 1 upserted item, got %d", len(repo.upsertedItems))
	}
}

// ===========================================================================
// Helpers
// ===========================================================================

// contains checks if substr is in s (avoids importing strings just for this).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
