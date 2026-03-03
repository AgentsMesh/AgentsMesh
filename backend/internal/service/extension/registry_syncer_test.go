package extension

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ===========================================================================
// McpRegistrySyncer - helper function tests
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

func newTestSyncer() *McpRegistrySyncer {
	return &McpRegistrySyncer{
		client: NewMcpRegistryClient(""),
		repo:   newMockExtensionRepo(),
	}
}

// --- applyPackageConfig ---

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
	if item.TransportType != extension.TransportTypeStdio {
		t.Errorf("expected transport 'stdio', got %q", item.TransportType)
	}
	if item.Command != "" {
		t.Errorf("expected empty command for unknown type, got %q", item.Command)
	}
}

// --- applyRemoteConfig ---

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
