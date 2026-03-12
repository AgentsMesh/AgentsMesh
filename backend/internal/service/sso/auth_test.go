package sso

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPasswordLoginAllowed_SystemAdminAlwaysAllowed(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOVal = true // SSO is enforced
	svc := newTestService(repo)

	allowed, err := svc.IsPasswordLoginAllowed(context.Background(), "admin@company.com", true)
	require.NoError(t, err)
	assert.True(t, allowed, "system admin should always be allowed password login")
}

func TestIsPasswordLoginAllowed_EnforcedSSOBlocksRegularUser(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOVal = true
	svc := newTestService(repo)

	allowed, err := svc.IsPasswordLoginAllowed(context.Background(), "user@company.com", false)
	require.NoError(t, err)
	assert.False(t, allowed, "regular user should be blocked when SSO is enforced")
}

func TestIsPasswordLoginAllowed_NoEnforcementAllowsAll(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOVal = false
	svc := newTestService(repo)

	allowed, err := svc.IsPasswordLoginAllowed(context.Background(), "user@company.com", false)
	require.NoError(t, err)
	assert.True(t, allowed, "should allow password login when SSO is not enforced")
}

func TestIsPasswordLoginAllowed_InvalidEmailAllowed(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOVal = true
	svc := newTestService(repo)

	// Email without @ should return empty domain → allowed
	allowed, err := svc.IsPasswordLoginAllowed(context.Background(), "noemail", false)
	require.NoError(t, err)
	assert.True(t, allowed, "invalid email should be allowed (can't extract domain)")
}

func TestIsPasswordLoginAllowed_FailOpenOnError(t *testing.T) {
	repo := newMockRepository()
	repo.hasEnforcedSSOErr = fmt.Errorf("database error")
	svc := newTestService(repo)

	allowed, err := svc.IsPasswordLoginAllowed(context.Background(), "user@company.com", false)
	require.NoError(t, err) // error is swallowed (fail-open)
	assert.True(t, allowed, "should fail-open on repository error to prevent lockout")
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{"user@company.com", "company.com"},
		{"User@COMPANY.COM", "company.com"},
		{"a@b.co", "b.co"},
		{"noemail", ""},
		{"", ""},
		{"@nodomain", "nodomain"},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equal(t, tt.want, extractDomain(tt.email))
		})
	}
}

func TestSSOProviderName(t *testing.T) {
	assert.Equal(t, "sso_oidc_42", SSOProviderName("oidc", 42))
	assert.Equal(t, "sso_saml_1", SSOProviderName("saml", 1))
	assert.Equal(t, "sso_ldap_100", SSOProviderName("ldap", 100))
}
