package extension

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ===========================================================================
// convertToMarketItem tests
// ===========================================================================

func TestConvertToMarketItem_NoName(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{Name: ""},
	}
	_, err := s.convertToMarketItem(entry, time.Now())
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "no name") {
		t.Errorf("expected 'no name' in error, got: %s", err.Error())
	}
}

func TestConvertToMarketItem_TitleFallback(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name:  "io.github.user/my-server",
			Title: "",
			Packages: []RegistryPackage{
				{RegistryType: "npm", Identifier: "@user/my-server"},
			},
		},
	}
	item, err := s.convertToMarketItem(entry, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Name != "my-server" {
		t.Errorf("expected name 'my-server', got %q", item.Name)
	}
}

func TestConvertToMarketItem_WithTitle(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name:  "io.github.user/my-server",
			Title: "My Cool Server",
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
			Name:     "test/npm-server",
			Packages: []RegistryPackage{{RegistryType: "npm", Identifier: "@test/npm-server"}},
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
			Name:     "test/pypi-server",
			Packages: []RegistryPackage{{RegistryType: "pypi", Identifier: "mcp-server-test"}},
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
			Name:     "test/oci-server",
			Packages: []RegistryPackage{{RegistryType: "oci", Identifier: "ghcr.io/test/server:latest"}},
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
			Packages: []RegistryPackage{{
				RegistryType: "npm",
				Identifier:   "@test/env-server",
				EnvironmentVariables: []RegistryEnvVar{
					{Name: "API_KEY", Description: "Your API key", IsRequired: true, IsSecret: true},
					{Name: "BASE_URL", Description: "Base URL for the API", IsRequired: false, IsSecret: false, Default: "https://api.example.com"},
				},
			}},
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
	if schema[0].Name != "API_KEY" || schema[0].Label != "Your API key" || !schema[0].Required || !schema[0].Sensitive {
		t.Errorf("unexpected first env var: %+v", schema[0])
	}
	if schema[1].Placeholder != "https://api.example.com" {
		t.Errorf("expected placeholder from default, got %q", schema[1].Placeholder)
	}
}

func TestConvertToMarketItem_RemoteSSE(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{
			Name:    "test/sse-server",
			Remotes: []RegistryRemote{{Type: "sse", URL: "https://sse.example.com/events"}},
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
			Name:    "test/http-server",
			Remotes: []RegistryRemote{{Type: "http", URL: "https://http.example.com/mcp"}},
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
			Remotes: []RegistryRemote{{
				Type: "sse",
				URL:  "https://sse.example.com/events",
				Headers: []RegistryHeader{
					{Name: "Authorization", Description: "Bearer token", IsRequired: true, IsSecret: true},
					{Name: "X-Custom", Description: "Custom header", Value: "custom-value", IsRequired: false, IsSecret: false},
				},
			}},
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
	if headers[0]["name"] != "Authorization" || headers[0]["required"] != true || headers[0]["sensitive"] != true {
		t.Errorf("unexpected first header: %v", headers[0])
	}
	if headers[1]["value"] != "custom-value" {
		t.Errorf("expected value 'custom-value', got %v", headers[1]["value"])
	}
}

func TestConvertToMarketItem_NoPackageOrRemote(t *testing.T) {
	s := newTestSyncer()
	entry := RegistryServerEntry{
		Server: RegistryServer{Name: "test/empty-server"},
	}
	_, err := s.convertToMarketItem(entry, time.Now())
	if err == nil {
		t.Fatal("expected error for no package or remote")
	}
	if !strings.Contains(err.Error(), "no usable package or remote") {
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
	if item.Command != "npx" {
		t.Errorf("expected command 'npx' (npm wins), got %q", item.Command)
	}
	if item.Category != "npm" {
		t.Errorf("expected category 'npm', got %q", item.Category)
	}
}
