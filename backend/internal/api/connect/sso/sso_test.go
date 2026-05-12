package ssoconnect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	domainUser "github.com/anthropics/agentsmesh/backend/internal/domain/user"
	ssov1 "github.com/anthropics/agentsmesh/proto/gen/go/sso/v1"
)

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- Discover input validation ---

func TestDiscover_EmptyEmail_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.Discover(context.Background(), connect.NewRequest(&ssov1.DiscoverRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestDiscover_InvalidEmail_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.Discover(context.Background(), connect.NewRequest(&ssov1.DiscoverRequest{
		Email: "not-an-email",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

// --- LdapAuth input validation ---

func TestLdapAuth_EmptyDomain_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.LdapAuth(context.Background(), connect.NewRequest(&ssov1.LdapAuthRequest{
		Username: "alice",
		Password: "p",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestLdapAuth_InvalidDomain_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.LdapAuth(context.Background(), connect.NewRequest(&ssov1.LdapAuthRequest{
		Domain:   "not a domain",
		Username: "alice",
		Password: "p",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestLdapAuth_EmptyCredentials_InvalidArgument(t *testing.T) {
	srv := NewServer(nil, nil)
	_, err := srv.LdapAuth(context.Background(), connect.NewRequest(&ssov1.LdapAuthRequest{
		Domain:   "acme.com",
		Username: "",
		Password: "p",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))

	_, err2 := srv.LdapAuth(context.Background(), connect.NewRequest(&ssov1.LdapAuthRequest{
		Domain:   "acme.com",
		Username: "alice",
		Password: "",
	}))
	require.Error(t, err2)
	require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err2))
}

// --- helpers ---

func TestExtractEmailDomain(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"alice@acme.com", "acme.com"},
		{"alice@ACME.com", "acme.com"},
		{"alice@sub.example.org", "sub.example.org"},
		{"no-at-sign", ""},
		{"", ""},
		{"a@b@c", "b@c"}, // SplitN with N=2 keeps any extra `@` in the domain
	}
	for _, tc := range cases {
		got := extractEmailDomain(tc.in)
		require.Equal(t, tc.want, got, "email=%q", tc.in)
	}
}

func TestValidateDomainFromRequest(t *testing.T) {
	good := []string{"acme.com", "ACME.COM", "sub.example.org", "  acme.com  "}
	for _, d := range good {
		got, err := validateDomainFromRequest(d)
		require.NoError(t, err, "domain=%q", d)
		require.NotEmpty(t, got)
		require.Equal(t, got, got) // sanity
	}
	bad := []string{"", "not a domain", "no-tld", "-bad.com", "bad-.com"}
	for _, d := range bad {
		_, err := validateDomainFromRequest(d)
		require.Error(t, err, "domain=%q should reject", d)
		require.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
	}
}

// --- convert helpers ---

func TestToProtoLdapAuthUser_WithName(t *testing.T) {
	name := "Alice Anderson"
	u := &domainUser.User{
		ID:       42,
		Email:    "alice@acme.com",
		Username: "alice",
		Name:     &name,
	}
	got := toProtoLdapAuthUser(u)
	require.NotNil(t, got)
	require.Equal(t, int64(42), got.GetId())
	require.Equal(t, "alice@acme.com", got.GetEmail())
	require.Equal(t, "alice", got.GetUsername())
	require.Equal(t, "Alice Anderson", got.GetName())
}

func TestToProtoLdapAuthUser_NoName(t *testing.T) {
	// LDAP record without displayName attribute — Name pointer is nil
	// on the domain side. Proto must round-trip as absent.
	u := &domainUser.User{
		ID:       7,
		Email:    "bob@acme.com",
		Username: "bob",
		Name:     nil,
	}
	got := toProtoLdapAuthUser(u)
	require.NotNil(t, got)
	require.Nil(t, got.Name, "absent name on User must remain nil on proto")
}

func TestToProtoLdapAuthUser_NilInput(t *testing.T) {
	require.Nil(t, toProtoLdapAuthUser(nil))
}

// --- service URL constants — pin against conventions §12 (canonical form) ---

func TestProcedureNamesMatchServiceName(t *testing.T) {
	require.Equal(t, "proto.sso.v1.SSOService", ServiceName)
	require.Equal(t, "/proto.sso.v1.SSOService/Discover", DiscoverProcedure)
	require.Equal(t, "/proto.sso.v1.SSOService/LdapAuth", LdapAuthProcedure)
}
