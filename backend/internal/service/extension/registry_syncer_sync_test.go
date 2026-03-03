package extension

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ===========================================================================
// Sync test mock
// ===========================================================================

// syncerMockRepo extends mockExtensionRepo with additional tracking for Sync tests.
type syncerMockRepo struct {
	mockExtensionRepo

	mu              sync.Mutex
	upsertedItems   []*extension.McpMarketItem
	batchUpsertFunc func(ctx context.Context, items []*extension.McpMarketItem) error
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

func (m *syncerMockRepo) BatchUpsertMcpMarketItems(ctx context.Context, items []*extension.McpMarketItem) error {
	if m.batchUpsertFunc != nil {
		return m.batchUpsertFunc(ctx, items)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upsertedItems = append(m.upsertedItems, items...)
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
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RegistryResponse{})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pages[idx])
	}))
}

// ===========================================================================
// Sync tests
// ===========================================================================

func TestSync_Success(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	pages := []RegistryResponse{{
		Servers: []RegistryServerEntry{
			{Server: RegistryServer{Name: "test/server1", Packages: []RegistryPackage{{RegistryType: "npm", Identifier: "@test/server1"}}}, Meta: activeMeta},
			{Server: RegistryServer{Name: "test/server2", Remotes: []RegistryRemote{{Type: "sse", URL: "https://example.com/sse"}}}, Meta: activeMeta},
		},
		Metadata: RegistryMetadata{Count: 2, NextCursor: ""},
	}}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	syncer := NewMcpRegistrySyncer(NewMcpRegistryClient(srv.URL), repo)

	if err := syncer.Sync(context.Background()); err != nil {
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
	syncer := NewMcpRegistrySyncer(NewMcpRegistryClient(srv.URL), repo)

	err := syncer.Sync(context.Background())
	if err == nil {
		t.Fatal("expected error when FetchAll fails")
	}
	if !strings.Contains(err.Error(), "fetch registry") {
		t.Errorf("expected 'fetch registry' in error, got: %s", err.Error())
	}
}

func TestSync_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := newSyncerMockRepo()
	syncer := NewMcpRegistrySyncer(NewMcpRegistryClient("http://unreachable.invalid"), repo)

	if err := syncer.Sync(ctx); err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

func TestSync_SkipInvalidEntries(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	pages := []RegistryResponse{{
		Servers: []RegistryServerEntry{
			{Server: RegistryServer{Name: ""}, Meta: activeMeta},                           // invalid: no name
			{Server: RegistryServer{Name: "test/no-config"}, Meta: activeMeta},              // invalid: no packages or remotes
			{Server: RegistryServer{Name: "test/valid", Packages: []RegistryPackage{{RegistryType: "npm", Identifier: "@test/valid"}}}, Meta: activeMeta},
		},
		Metadata: RegistryMetadata{Count: 3},
	}}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	syncer := NewMcpRegistrySyncer(NewMcpRegistryClient(srv.URL), repo)

	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.upsertedItems) != 1 {
		t.Fatalf("expected 1 upserted item, got %d", len(repo.upsertedItems))
	}
	if repo.upsertedItems[0].RegistryName != "test/valid" {
		t.Errorf("expected 'test/valid', got %q", repo.upsertedItems[0].RegistryName)
	}
}

func TestSync_BatchUpsertError(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	pages := []RegistryResponse{{
		Servers: []RegistryServerEntry{
			{Server: RegistryServer{Name: "test/server1", Packages: []RegistryPackage{{RegistryType: "npm", Identifier: "@test/server1"}}}, Meta: activeMeta},
		},
		Metadata: RegistryMetadata{Count: 1},
	}}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	repo.batchUpsertFunc = func(_ context.Context, _ []*extension.McpMarketItem) error {
		return errors.New("db write error")
	}

	syncer := NewMcpRegistrySyncer(NewMcpRegistryClient(srv.URL), repo)
	err := syncer.Sync(context.Background())
	if err == nil {
		t.Fatal("expected error when batch upsert fails")
	}
	if !strings.Contains(err.Error(), "batch upsert") {
		t.Errorf("expected 'batch upsert' in error, got: %s", err.Error())
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.deactivateCalls) != 0 {
		t.Errorf("expected 0 deactivate calls when upsert fails, got %d", len(repo.deactivateCalls))
	}
}

func TestSync_DeactivateError(t *testing.T) {
	activeMeta := json.RawMessage(`{"io.modelcontextprotocol.registry/official": {"isLatest": true, "status": "active"}}`)
	pages := []RegistryResponse{{
		Servers: []RegistryServerEntry{
			{Server: RegistryServer{Name: "test/server1", Packages: []RegistryPackage{{RegistryType: "npm", Identifier: "@test/server1"}}}, Meta: activeMeta},
		},
		Metadata: RegistryMetadata{Count: 1},
	}}

	srv := newRegistryServer(t, pages)
	defer srv.Close()

	repo := newSyncerMockRepo()
	repo.deactivateFunc = func(_ context.Context, _ string, _ []string) (int64, error) {
		return 0, errors.New("deactivation failed")
	}

	syncer := NewMcpRegistrySyncer(NewMcpRegistryClient(srv.URL), repo)

	// Sync should NOT return an error even when deactivation fails
	if err := syncer.Sync(context.Background()); err != nil {
		t.Fatalf("expected no error (deactivation failure is warn-only), got: %v", err)
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.upsertedItems) != 1 {
		t.Errorf("expected 1 upserted item, got %d", len(repo.upsertedItems))
	}
}
