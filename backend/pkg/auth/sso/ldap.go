package sso

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	ldapv3 "github.com/go-ldap/ldap/v3"
)

// ldapConnectTimeout is the maximum time to wait for an LDAP TCP connection.
const ldapConnectTimeout = 10 * time.Second

// LDAPConfig holds LDAP provider configuration
type LDAPConfig struct {
	Host         string
	Port         int
	UseTLS       bool
	BindDN       string
	BindPassword string
	BaseDN       string
	UserFilter   string // e.g., "(uid={{username}})" or "(sAMAccountName={{username}})"
	EmailAttr    string // default: "mail"
	NameAttr     string // default: "cn"
	UsernameAttr string // default: "uid"
}

// LDAPProvider implements Provider for LDAP authentication
type LDAPProvider struct {
	config *LDAPConfig
}

// NewLDAPProvider creates a new LDAP provider
func NewLDAPProvider(cfg *LDAPConfig) (*LDAPProvider, error) {
	if cfg.Host == "" || cfg.BaseDN == "" {
		return nil, fmt.Errorf("%w: missing LDAP host or base DN", ErrInvalidConfig)
	}

	// Set defaults
	if cfg.Port == 0 {
		if cfg.UseTLS {
			cfg.Port = 636
		} else {
			cfg.Port = 389
		}
	}
	if cfg.EmailAttr == "" {
		cfg.EmailAttr = "mail"
	}
	if cfg.NameAttr == "" {
		cfg.NameAttr = "cn"
	}
	if cfg.UsernameAttr == "" {
		cfg.UsernameAttr = "uid"
	}
	if cfg.UserFilter == "" {
		cfg.UserFilter = "(uid={{username}})"
	}

	return &LDAPProvider{config: cfg}, nil
}

// GetAuthURL is not supported for LDAP
func (p *LDAPProvider) GetAuthURL(_ context.Context, _ string) (string, error) {
	return "", ErrNotSupported
}

// HandleCallback is not supported for LDAP
func (p *LDAPProvider) HandleCallback(_ context.Context, _ map[string]string) (*UserInfo, error) {
	return nil, ErrNotSupported
}

// Authenticate performs LDAP bind authentication
func (p *LDAPProvider) Authenticate(_ context.Context, username, password string) (*UserInfo, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("%w: username and password required", ErrAuthFailed)
	}

	// Connect to LDAP
	conn, err := p.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	// Service account bind (to search for user)
	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return nil, fmt.Errorf("service account bind failed: %w", err)
		}
	}

	// Search for user
	filter := strings.ReplaceAll(p.config.UserFilter, "{{username}}", ldapv3.EscapeFilter(username))
	searchReq := ldapv3.NewSearchRequest(
		p.config.BaseDN,
		ldapv3.ScopeWholeSubtree,
		ldapv3.NeverDerefAliases,
		1,  // size limit
		30, // time limit (seconds)
		false,
		filter,
		[]string{"dn", p.config.EmailAttr, p.config.NameAttr, p.config.UsernameAttr},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("%w: user not found", ErrAuthFailed)
	}
	if len(result.Entries) > 1 {
		return nil, fmt.Errorf("%w: multiple users found", ErrAuthFailed)
	}

	entry := result.Entries[0]

	// User bind (verify password)
	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", ErrAuthFailed)
	}

	// Extract user info
	email := entry.GetAttributeValue(p.config.EmailAttr)
	if email == "" {
		return nil, fmt.Errorf("%w: email attribute %q is empty for user %s", ErrAuthFailed, p.config.EmailAttr, entry.DN)
	}

	return &UserInfo{
		ExternalID: entry.DN,
		Email:      email,
		Username:   entry.GetAttributeValue(p.config.UsernameAttr),
		Name:       entry.GetAttributeValue(p.config.NameAttr),
	}, nil
}

// connect establishes connection to LDAP server
func (p *LDAPProvider) connect() (*ldapv3.Conn, error) {
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	if p.config.UseTLS {
		return ldapv3.DialURL(
			fmt.Sprintf("ldaps://%s", addr),
			ldapv3.DialWithDialer(&net.Dialer{Timeout: ldapConnectTimeout}),
			ldapv3.DialWithTLSConfig(&tls.Config{
				ServerName: p.config.Host,
				MinVersion: tls.VersionTLS12,
			}),
		)
	}

	// UseTLS=false means plaintext LDAP — no StartTLS upgrade.
	// Admin explicitly chose this for LDAP servers that don't support TLS
	// (e.g., internal networks, development environments).
	return ldapv3.DialURL(
		fmt.Sprintf("ldap://%s", addr),
		ldapv3.DialWithDialer(&net.Dialer{Timeout: ldapConnectTimeout}),
	)
}

// TestConnection tests the LDAP connection and service bind
func (p *LDAPProvider) TestConnection() error {
	conn, err := p.connect()
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return fmt.Errorf("service bind failed: %w", err)
		}
	}

	return nil
}
