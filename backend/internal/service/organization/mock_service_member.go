package organization

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

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
func (m *MockService) ListMembers(ctx context.Context, orgID int64) ([]*organization.Member, error) {
	if m.ListMembersErr != nil {
		return nil, m.ListMembersErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*organization.Member
	if members, ok := m.members[orgID]; ok {
		for userID, member := range members {
			memberCopy := *member
			memberCopy.User = &user.User{ID: userID}
			result = append(result, &memberCopy)
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
