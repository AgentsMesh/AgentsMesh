package policy

type Subject struct {
	OrgID  int64
	UserID int64
	Role   string // "owner", "admin", "member", "apikey"
}

func NewSubject(orgID, userID int64, role string) Subject {
	return Subject{OrgID: orgID, UserID: userID, Role: role}
}

func (s Subject) isAdmin() bool  { return s.Role == "owner" || s.Role == "admin" }
func (s Subject) isAPIKey() bool { return s.Role == "apikey" }

type ResourceContext struct {
	OrgID          int64
	OwnerID        int64
	Visibility     string
	GrantedUserIDs []int64
}

func PodResource(orgID, createdByID int64) ResourceContext {
	return ResourceContext{OrgID: orgID, OwnerID: createdByID}
}

func VisibleResource(orgID int64, ownerIDPtr *int64, visibility string) ResourceContext {
	var ownerID int64
	if ownerIDPtr != nil {
		ownerID = *ownerIDPtr
	}
	return ResourceContext{OrgID: orgID, OwnerID: ownerID, Visibility: visibility}
}

func (rc ResourceContext) WithGrants(userIDs []int64) ResourceContext {
	rc.GrantedUserIDs = userIDs
	return rc
}

func (rc ResourceContext) isGranted(userID int64) bool {
	for _, id := range rc.GrantedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

const (
	VisibilityOrganization = "organization"
	VisibilityPrivate      = "private"
)
type ReadAccess int

const (
	ReadOrgOpen    ReadAccess = iota // any org member
	ReadOwnerOnly                    // members see own; admins/apikeys see all
	ReadVisibility                   // Visibility field controls; no admin bypass
)

type WriteAccess int

const (
	WriteOrgOpen      WriteAccess = iota // any org member
	WriteCreatorAdmin                    // creator or admin
	WriteAdminOnly                       // admin only
)

type ResourcePolicy struct {
	Read  ReadAccess
	Write WriteAccess
}

func (p ResourcePolicy) AllowRead(s Subject, res ResourceContext) bool {
	if res.OrgID != s.OrgID {
		return false
	}
	switch p.Read {
	case ReadOrgOpen:
		return true
	case ReadOwnerOnly:
		return s.isAdmin() || s.isAPIKey() || res.OwnerID == s.UserID || res.isGranted(s.UserID)
	case ReadVisibility:
		if res.Visibility == VisibilityPrivate {
			return res.OwnerID == s.UserID || res.isGranted(s.UserID)
		}
		return true
	}
	return false
}

func (p ResourcePolicy) AllowWrite(s Subject, res ResourceContext) bool {
	if res.OrgID != s.OrgID {
		return false
	}
	switch p.Write {
	case WriteOrgOpen:
		return true
	case WriteCreatorAdmin:
		return s.isAdmin() || res.OwnerID == s.UserID || res.isGranted(s.UserID)
	case WriteAdminOnly:
		return s.isAdmin()
	}
	return false
}

func AllowAdmin(s Subject, orgID int64) bool {
	return s.OrgID == orgID && s.isAdmin()
}

type ListFilter struct {
	OwnerOnly int64
	VisibilityUserID int64
	GrantUserID int64
}

func (p ResourcePolicy) ListFilter(s Subject) ListFilter {
	switch p.Read {
	case ReadOwnerOnly:
		if !s.isAdmin() && !s.isAPIKey() {
			return ListFilter{OwnerOnly: s.UserID, GrantUserID: s.UserID}
		}
		return ListFilter{}
	case ReadVisibility:
		return ListFilter{VisibilityUserID: s.UserID, GrantUserID: s.UserID}
	}
	return ListFilter{}
}
