package sso

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ sso.Repository = (*mockRepository)(nil)

// mockRepository is an in-memory mock of sso.Repository for unit tests.
type mockRepository struct {
	mu      sync.Mutex
	configs map[int64]*sso.Config // keyed by ID
	nextID  int64

	// Error injection
	createErr          error
	getByIDErr         error
	getByDomainErr     error
	listByDomainErr    error
	getEnabledErr      error
	listErr            error
	updateErr          error
	deleteErr          error
	hasEnforcedSSOErr  error
	hasEnforcedSSOVal  bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		configs: make(map[int64]*sso.Config),
		nextID:  1,
	}
}

func (m *mockRepository) Create(_ context.Context, cfg *sso.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	// Check duplicate domain+protocol
	for _, c := range m.configs {
		if c.Domain == cfg.Domain && c.Protocol == cfg.Protocol {
			return nil // caller checks via GetByDomainAndProtocol first
		}
	}
	cfg.ID = m.nextID
	m.nextID++
	clone := *cfg
	m.configs[cfg.ID] = &clone
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id int64) (*sso.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	cfg, ok := m.configs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	clone := *cfg
	return &clone, nil
}

func (m *mockRepository) GetByDomainAndProtocol(_ context.Context, domain string, protocol sso.Protocol) (*sso.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getByDomainErr != nil {
		return nil, m.getByDomainErr
	}
	for _, cfg := range m.configs {
		if cfg.Domain == domain && cfg.Protocol == protocol {
			clone := *cfg
			return &clone, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockRepository) ListByDomain(_ context.Context, domain string) ([]*sso.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listByDomainErr != nil {
		return nil, m.listByDomainErr
	}
	var result []*sso.Config
	for _, cfg := range m.configs {
		if cfg.Domain == domain {
			clone := *cfg
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (m *mockRepository) GetEnabledByDomain(_ context.Context, domain string) ([]*sso.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getEnabledErr != nil {
		return nil, m.getEnabledErr
	}
	var result []*sso.Config
	for _, cfg := range m.configs {
		if cfg.Domain == domain && cfg.IsEnabled {
			clone := *cfg
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (m *mockRepository) List(_ context.Context, _ *sso.ListQuery, offset, limit int) ([]*sso.Config, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var all []*sso.Config
	for _, cfg := range m.configs {
		clone := *cfg
		all = append(all, &clone)
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *mockRepository) Update(_ context.Context, id int64, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.configs[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	// In a real implementation, we'd apply updates to the config fields.
	// For unit tests, we only need to verify the update map is correct.
	return nil
}

func (m *mockRepository) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.configs[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	delete(m.configs, id)
	return nil
}

func (m *mockRepository) HasEnforcedSSO(_ context.Context, _ string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hasEnforcedSSOErr != nil {
		return false, m.hasEnforcedSSOErr
	}
	return m.hasEnforcedSSOVal, nil
}

// seedConfig inserts a config directly into the mock store (bypasses Create validation).
func (m *mockRepository) seedConfig(cfg *sso.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.ID == 0 {
		cfg.ID = m.nextID
		m.nextID++
	}
	clone := *cfg
	m.configs[cfg.ID] = &clone
}
