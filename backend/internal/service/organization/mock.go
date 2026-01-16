package organization

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
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
		SubscriptionPlan:   "free",
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

// AddMember implements Interface.
func (m *MockService) AddMember(ctx context.Context, orgID, userID int64, role string) error {
	if m.AddMemberErr != nil {
		return m.AddMemberErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.AddedMembers = append(m.AddedMembers, memberOp{OrgID: orgID, UserID: userID, Role: role})

	if m.members[orgID] == nil {
		m.members[orgID] = make(map[int64]*organization.Member)
	}
	m.members[orgID][userID] = &organization.Member{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
	return nil
}

// RemoveMember implements Interface.
func (m *MockService) RemoveMember(ctx context.Context, orgID, userID int64) error {
	if m.RemoveMemberErr != nil {
		return m.RemoveMemberErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if owner
	if member, ok := m.members[orgID][userID]; ok && member.Role == organization.RoleOwner {
		return ErrCannotRemoveOwner
	}

	m.RemovedMembers = append(m.RemovedMembers, memberOp{OrgID: orgID, UserID: userID})
	delete(m.members[orgID], userID)
	return nil
}

// UpdateMemberRole implements Interface.
func (m *MockService) UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error {
	if m.UpdateMemberRoleErr != nil {
		return m.UpdateMemberRoleErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if member, ok := m.members[orgID][userID]; ok {
		member.Role = role
	}
	return nil
}

// GetMember implements Interface.
func (m *MockService) GetMember(ctx context.Context, orgID, userID int64) (*organization.Member, error) {
	if m.GetMemberErr != nil {
		return nil, m.GetMemberErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if members, ok := m.members[orgID]; ok {
		if member, ok := members[userID]; ok {
			return member, nil
		}
	}
	return nil, ErrOrganizationNotFound
}

// ListMembers implements Interface.
func (m *MockService) ListMembers(ctx context.Context, orgID int64) ([]*user.User, error) {
	if m.ListMembersErr != nil {
		return nil, m.ListMembersErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*user.User
	if members, ok := m.members[orgID]; ok {
		for userID := range members {
			result = append(result, &user.User{ID: userID})
		}
	}
	return result, nil
}

// IsAdmin implements Interface.
func (m *MockService) IsAdmin(ctx context.Context, orgID, userID int64) (bool, error) {
	if m.IsAdminErr != nil {
		return false, m.IsAdminErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if members, ok := m.members[orgID]; ok {
		if member, ok := members[userID]; ok {
			return member.Role == organization.RoleOwner || member.Role == organization.RoleAdmin, nil
		}
	}
	return false, nil
}

// IsOwner implements Interface.
func (m *MockService) IsOwner(ctx context.Context, orgID, userID int64) (bool, error) {
	if m.IsOwnerErr != nil {
		return false, m.IsOwnerErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if members, ok := m.members[orgID]; ok {
		if member, ok := members[userID]; ok {
			return member.Role == organization.RoleOwner, nil
		}
	}
	return false, nil
}

// IsMember implements Interface.
func (m *MockService) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	if m.IsMemberErr != nil {
		return false, m.IsMemberErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if members, ok := m.members[orgID]; ok {
		_, isMember := members[userID]
		return isMember, nil
	}
	return false, nil
}

// GetUserRole implements Interface.
func (m *MockService) GetUserRole(ctx context.Context, orgID, userID int64) (string, error) {
	if m.GetUserRoleErr != nil {
		return "", m.GetUserRoleErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if members, ok := m.members[orgID]; ok {
		if member, ok := members[userID]; ok {
			return member.Role, nil
		}
	}
	return "", ErrOrganizationNotFound
}

// GetMemberRole implements Interface.
func (m *MockService) GetMemberRole(ctx context.Context, orgID, userID int64) (string, error) {
	return m.GetUserRole(ctx, orgID, userID)
}

// --- Test Helper Methods ---

// AddOrg adds an organization to the mock storage.
func (m *MockService) AddOrg(org *organization.Organization) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if org.ID == 0 {
		org.ID = m.nextID
		m.nextID++
	}
	m.orgs[org.ID] = org
	m.orgsBySlug[org.Slug] = org
}

// SetMember sets a member for an organization.
func (m *MockService) SetMember(orgID, userID int64, role string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.members[orgID] == nil {
		m.members[orgID] = make(map[int64]*organization.Member)
	}
	m.members[orgID][userID] = &organization.Member{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
}

// GetOrgs returns all organizations (thread-safe).
func (m *MockService) GetOrgs() []*organization.Organization {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*organization.Organization, 0, len(m.orgs))
	for _, org := range m.orgs {
		result = append(result, org)
	}
	return result
}

// Reset clears all data.
func (m *MockService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.orgs = make(map[int64]*organization.Organization)
	m.orgsBySlug = make(map[string]*organization.Organization)
	m.members = make(map[int64]map[int64]*organization.Member)
	m.nextID = 1
	m.CreatedOrgs = nil
	m.UpdatedOrgs = nil
	m.DeletedOrgIDs = nil
	m.AddedMembers = nil
	m.RemovedMembers = nil
}

// Ensure MockService implements Interface
var _ Interface = (*MockService)(nil)
