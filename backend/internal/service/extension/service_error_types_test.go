package extension

import (
	"context"
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
)

// ---------------------------------------------------------------------------
// Tests: StandardErrorTypes
// ---------------------------------------------------------------------------

func TestStandardErrorTypes(t *testing.T) {
	t.Run("validateScope_returns_ErrInvalidScope", func(t *testing.T) {
		err := validateScope("bad")
		if !errors.Is(err, ErrInvalidScope) {
			t.Errorf("expected errors.Is(err, ErrInvalidScope), got: %v", err)
		}
	})

	t.Run("SyncSkillRegistry_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.SyncSkillRegistry(context.Background(), 1, 999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("DeleteSkillRegistry_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getSkillRegistryFn: func(_ context.Context, id int64) (*extension.SkillRegistry, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		err := svc.DeleteSkillRegistry(context.Background(), 1, 999)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UpdateSkill_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.UpdateSkill(context.Background(), 1, 0, 999, 100, "admin", nil, nil)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UninstallSkill_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledSkillFn: func(_ context.Context, id int64) (*extension.InstalledSkill, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		err := svc.UninstallSkill(context.Background(), 1, 0, 999, 100, "admin")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UpdateMcpServer_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.UpdateMcpServer(context.Background(), 1, 0, 999, 100, "admin", nil, nil)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("UninstallMcpServer_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getInstalledMcpServerFn: func(_ context.Context, id int64) (*extension.InstalledMcpServer, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		err := svc.UninstallMcpServer(context.Background(), 1, 0, 999, 100, "admin")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("InstallSkillFromMarket_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getSkillMarketItemFn: func(_ context.Context, id int64) (*extension.SkillMarketItem, error) {
				return nil, errors.New("record not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.InstallSkillFromMarket(context.Background(), 1, 2, 3, 999, "org")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})

	t.Run("InstallMcpFromMarket_notfound_returns_ErrNotFound", func(t *testing.T) {
		repo := &svcMockRepo{
			getMcpMarketItemFn: func(_ context.Context, id int64) (*extension.McpMarketItem, error) {
				return nil, errors.New("record not found")
			},
		}
		svc := newTestService(repo, &svcMockStorage{}, nil)
		_, err := svc.InstallMcpFromMarket(context.Background(), 1, 2, 3, 999, nil, "org")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}
