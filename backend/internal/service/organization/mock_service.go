package organization

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// MockService is a mock implementation of Interface for testing.
type MockService struct {
	mu sync.RWMutex

	// In-memory storage
	orgs       map[int64]*organization.Organization
	orgsBySlug map[string]*organization.Organization
	members    map[int64]map[int64]*organization.Member // orgID -> userID -> member
	nextID     int64

	// Configurable error responses
	CreateErr           error
	GetByIDErr          error
	GetBySlugErr        error
	UpdateErr           error
	DeleteErr           error
	ListByUserErr       error
	AddMemberErr        error
	RemoveMemberErr     error
	UpdateMemberRoleErr error
	GetMemberErr        error
	ListMembersErr      error
	IsAdminErr          error
	IsOwnerErr          error
	IsMemberErr         error
	GetUserRoleErr      error

	// Captured calls for verification
	CreatedOrgs    []*CreateRequest
	UpdatedOrgs    []map[string]interface{}
	DeletedOrgIDs  []int64
	AddedMembers   []memberOp
	RemovedMembers []memberOp
}

type memberOp struct {
	OrgID  int64
	UserID int64
	Role   string
}

// NewMockService creates a new mock organization service for testing.
func NewMockService() *MockService {
	return &MockService{
		orgs:       make(map[int64]*organization.Organization),
		orgsBySlug: make(map[string]*organization.Organization),
		members:    make(map[int64]map[int64]*organization.Member),
		nextID:     1,
	}
}

// Create implements Interface.
func (m *MockService) Create(ctx context.Context, ownerID int64, req *CreateRequest) (*organization.Organization, error) {
	if m.CreateErr != nil {
		return nil, m.CreateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if slug exists
	if _, exists := m.orgsBySlug[req.Slug]; exists {
		return nil, ErrSlugAlreadyExists
	}

	m.CreatedOrgs = append(m.CreatedOrgs, req)

	org := &organization.Organization{
		ID:                 m.nextID,
		Name:               req.Name,
		Slug:               req.Slug,
		SubscriptionPlan:   "based",
		SubscriptionStatus: "active",
	}
	if req.LogoURL != "" {
		org.LogoURL = &req.LogoURL
	}

	m.orgs[m.nextID] = org
	m.orgsBySlug[req.Slug] = org

	// Add owner as member
	if m.members[m.nextID] == nil {
		m.members[m.nextID] = make(map[int64]*organization.Member)
	}
	m.members[m.nextID][ownerID] = &organization.Member{
		OrganizationID: m.nextID,
		UserID:         ownerID,
		Role:           organization.RoleOwner,
	}

	m.nextID++
	return org, nil
}

// GetByID implements Interface.
func (m *MockService) GetByID(ctx context.Context, id int64) (*organization.Organization, error) {
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if org, ok := m.orgs[id]; ok {
		return org, nil
	}
	return nil, ErrOrganizationNotFound
}

// GetBySlug implements Interface.
func (m *MockService) GetBySlug(ctx context.Context, slug string) (middleware.OrganizationGetter, error) {
	if m.GetBySlugErr != nil {
		return nil, m.GetBySlugErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if org, ok := m.orgsBySlug[slug]; ok {
		return org, nil
	}
	return nil, ErrOrganizationNotFound
}

// GetOrgBySlug implements Interface.
func (m *MockService) GetOrgBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	if m.GetBySlugErr != nil {
		return nil, m.GetBySlugErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if org, ok := m.orgsBySlug[slug]; ok {
		return org, nil
	}
	return nil, ErrOrganizationNotFound
}

// Update implements Interface.
func (m *MockService) Update(ctx context.Context, id int64, updates map[string]interface{}) (*organization.Organization, error) {
	if m.UpdateErr != nil {
		return nil, m.UpdateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdatedOrgs = append(m.UpdatedOrgs, updates)

	org, ok := m.orgs[id]
	if !ok {
		return nil, ErrOrganizationNotFound
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		org.Name = name
	}

	return org, nil
}

// Delete implements Interface.
func (m *MockService) Delete(ctx context.Context, id int64) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeletedOrgIDs = append(m.DeletedOrgIDs, id)

	if org, ok := m.orgs[id]; ok {
		delete(m.orgsBySlug, org.Slug)
	}
	delete(m.orgs, id)
	delete(m.members, id)
	return nil
}

// ListByUser implements Interface.
func (m *MockService) ListByUser(ctx context.Context, userID int64) ([]*organization.Organization, error) {
	if m.ListByUserErr != nil {
		return nil, m.ListByUserErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*organization.Organization
	for orgID, members := range m.members {
		if _, isMember := members[userID]; isMember {
			if org, ok := m.orgs[orgID]; ok {
				result = append(result, org)
			}
		}
	}
	return result, nil
}

// Ensure MockService implements Interface
var _ Interface = (*MockService)(nil)
