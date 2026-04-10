package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// --- Subject helpers ---

func subjectMember(orgID, userID int64) Subject {
	return Subject{OrgID: orgID, UserID: userID, Role: "member"}
}

func subjectAdmin(orgID, userID int64) Subject {
	return Subject{OrgID: orgID, UserID: userID, Role: "admin"}
}

func subjectOwnerRole(orgID, userID int64) Subject {
	return Subject{OrgID: orgID, UserID: userID, Role: "owner"}
}

func subjectAPIKey(orgID, userID int64) Subject {
	return Subject{OrgID: orgID, UserID: userID, Role: "apikey"}
}

// --- From() ---

func TestFrom(t *testing.T) {
	tc := &middleware.TenantContext{
		OrganizationID: 5,
		UserID:         7,
		UserRole:       "admin",
	}
	s := From(tc)
	assert.Equal(t, int64(5), s.OrgID)
	assert.Equal(t, int64(7), s.UserID)
	assert.Equal(t, "admin", s.Role)
}

// --- AllowRead: ReadOwnerOnly (PodPolicy) ---

func TestAllowRead_OwnerOnly(t *testing.T) {
	p := PodPolicy
	res := ResourceContext{OrgID: 1, OwnerID: 10}

	cases := []struct {
		name    string
		subject Subject
		want    bool
	}{
		{"member reads own pod", subjectMember(1, 10), true},
		{"member reads other pod", subjectMember(1, 99), false},
		{"admin reads other pod", subjectAdmin(1, 99), true},
		{"owner role reads other pod", subjectOwnerRole(1, 99), true},
		{"apikey reads any pod", subjectAPIKey(1, 42), true},
		{"wrong org", subjectAdmin(2, 10), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, p.AllowRead(tc.subject, res))
		})
	}
}

// --- AllowRead: ReadVisibility (RunnerPolicy) ---

func TestAllowRead_Visibility_Organization(t *testing.T) {
	p := RunnerPolicy
	res := ResourceContext{OrgID: 1, OwnerID: 10, Visibility: "organization"}

	cases := []struct {
		name    string
		subject Subject
		want    bool
	}{
		{"member reads org runner", subjectMember(1, 42), true},
		{"admin reads org runner", subjectAdmin(1, 99), true},
		{"apikey reads org runner", subjectAPIKey(1, 55), true},
		{"wrong org", subjectMember(2, 10), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, p.AllowRead(tc.subject, res))
		})
	}
}

func TestAllowRead_Visibility_Private(t *testing.T) {
	p := RunnerPolicy
	res := ResourceContext{OrgID: 1, OwnerID: 10, Visibility: "private"}

	cases := []struct {
		name    string
		subject Subject
		want    bool
	}{
		{"owner reads private runner", subjectMember(1, 10), true},
		{"member reads private runner", subjectMember(1, 42), false},
		// admin is also denied — ReadVisibility has no admin bypass (intentional)
		{"admin reads private runner", subjectAdmin(1, 99), false},
		{"apikey reads private runner", subjectAPIKey(1, 55), false},
		{"wrong org", subjectAdmin(2, 10), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, p.AllowRead(tc.subject, res))
		})
	}
}

// --- AllowRead: explicit GrantedUserIDs ---

func TestAllowRead_ExplicitGrant(t *testing.T) {
	p := PodPolicy
	// Pod owned by user 10; user 20 explicitly granted.
	res := ResourceContext{OrgID: 1, OwnerID: 10, GrantedUserIDs: []int64{20}}

	assert.True(t, p.AllowRead(subjectMember(1, 10), res), "owner may read")
	assert.True(t, p.AllowRead(subjectMember(1, 20), res), "granted user may read")
	assert.False(t, p.AllowRead(subjectMember(1, 30), res), "non-granted member may not read")
}

// --- AllowWrite: WriteCreatorAdmin (PodPolicy) ---

func TestAllowWrite_CreatorAdmin(t *testing.T) {
	p := PodPolicy
	res := ResourceContext{OrgID: 1, OwnerID: 10}

	cases := []struct {
		name    string
		subject Subject
		want    bool
	}{
		{"creator writes own pod", subjectMember(1, 10), true},
		{"member writes other pod", subjectMember(1, 42), false},
		{"admin writes other pod", subjectAdmin(1, 99), true},
		{"owner role writes other pod", subjectOwnerRole(1, 99), true},
		{"wrong org", subjectAdmin(2, 10), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, p.AllowWrite(tc.subject, res))
		})
	}
}

// --- AllowWrite: WriteAdminOnly (RunnerPolicy) ---

func TestAllowWrite_AdminOnly(t *testing.T) {
	p := RunnerPolicy
	res := ResourceContext{OrgID: 1, OwnerID: 10}

	cases := []struct {
		name    string
		subject Subject
		want    bool
	}{
		{"admin writes runner", subjectAdmin(1, 99), true},
		{"owner role writes runner", subjectOwnerRole(1, 99), true},
		// member is denied even if they are the registered owner
		{"creator member writes runner", subjectMember(1, 10), false},
		{"other member writes runner", subjectMember(1, 42), false},
		{"wrong org", subjectAdmin(2, 10), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, p.AllowWrite(tc.subject, res))
		})
	}
}

// --- FilterList ---

func TestFilterList(t *testing.T) {
	p := PodPolicy

	assert.Equal(t, int64(7), p.FilterList(subjectMember(1, 7)), "member gets their own userID")
	assert.Equal(t, int64(0), p.FilterList(subjectAdmin(1, 7)), "admin gets 0 (no filter)")
	assert.Equal(t, int64(0), p.FilterList(subjectOwnerRole(1, 7)), "owner role gets 0 (no filter)")
	assert.Equal(t, int64(0), p.FilterList(subjectAPIKey(1, 7)), "apikey gets 0 (no filter)")
}

func TestFilterList_OrgOpenPolicy(t *testing.T) {
	p := ResourcePolicy{Read: ReadOrgOpen, Write: WriteOrgOpen}
	// ReadOrgOpen never filters by owner regardless of role
	assert.Equal(t, int64(0), p.FilterList(subjectMember(1, 7)))
}
