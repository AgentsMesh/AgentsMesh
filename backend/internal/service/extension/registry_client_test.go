package extension

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain status code 500, got: %s", err.Error())
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
	if !strings.Contains(err.Error(), "decode response") {
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
	if !strings.Contains(err.Error(), "fetch page 0") {
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
