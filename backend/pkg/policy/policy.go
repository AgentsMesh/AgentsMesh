package policy

import "github.com/anthropics/agentsmesh/backend/internal/middleware"

// Subject is the acting user extracted from tenant context.
type Subject struct {
	OrgID  int64
	UserID int64
	Role   string // "owner", "admin", "member", "apikey"
}

// From constructs a Subject from a TenantContext.
func From(t *middleware.TenantContext) Subject {
	return Subject{OrgID: t.OrganizationID, UserID: t.UserID, Role: t.UserRole}
}

func (s Subject) isAdmin() bool  { return s.Role == "owner" || s.Role == "admin" }
func (s Subject) isAPIKey() bool { return s.Role == "apikey" }

// ResourceContext carries the access-relevant fields of a resource instance.
type ResourceContext struct {
	OrgID          int64
	OwnerID        int64   // 0 = no owner concept (admin-only resources)
	Visibility     string  // "" | "organization" | "private"
	GrantedUserIDs []int64 // explicit per-instance grants (Phase 2; pass nil for now)
}

// ReadAccess enumerates the read access modes a policy may declare.
type ReadAccess int

const (
	ReadOrgOpen    ReadAccess = iota // any org member may read
	ReadOwnerOnly                    // members see own; admins/apikeys see all
	ReadVisibility                   // Visibility field controls access; no admin bypass
)

// WriteAccess enumerates the write access modes a policy may declare.
type WriteAccess int

const (
	WriteOrgOpen      WriteAccess = iota // any org member may write
	WriteCreatorAdmin                    // creator or admin/owner may write
	WriteAdminOnly                       // only admin/owner may write
)

// ResourcePolicy declares the access rules for one resource type.
type ResourcePolicy struct {
	Read  ReadAccess
	Write WriteAccess
}

// AllowRead returns true if subject may read the resource.
func (p ResourcePolicy) AllowRead(s Subject, res ResourceContext) bool {
	if res.OrgID != s.OrgID {
		return false
	}
	// Explicit grant check (Phase 2: populated from resource_grants table).
	for _, id := range res.GrantedUserIDs {
		if id == s.UserID {
			return true
		}
	}
	switch p.Read {
	case ReadOrgOpen:
		return true
	case ReadOwnerOnly:
		return s.isAdmin() || s.isAPIKey() || res.OwnerID == s.UserID
	case ReadVisibility:
		if res.Visibility == "private" {
			return res.OwnerID == s.UserID
		}
		return true
	}
	return false
}

// AllowWrite returns true if subject may mutate the resource.
func (p ResourcePolicy) AllowWrite(s Subject, res ResourceContext) bool {
	if res.OrgID != s.OrgID {
		return false
	}
	switch p.Write {
	case WriteOrgOpen:
		return true
	case WriteCreatorAdmin:
		return s.isAdmin() || res.OwnerID == s.UserID
	case WriteAdminOnly:
		return s.isAdmin()
	}
	return false
}

// FilterList returns a non-zero ownerID to narrow list queries, or 0 for no filter.
// Pass the returned value as a createdByID/ownerID filter to the repository.
func (p ResourcePolicy) FilterList(s Subject) int64 {
	if p.Read == ReadOwnerOnly && !s.isAdmin() && !s.isAPIKey() {
		return s.UserID
	}
	return 0
}
