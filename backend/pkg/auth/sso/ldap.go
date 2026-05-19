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

const ldapConnectTimeout = 10 * time.Second

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

type LDAPProvider struct {
	config *LDAPConfig
}

func NewLDAPProvider(cfg *LDAPConfig) (*LDAPProvider, error) {
	if cfg.Host == "" || cfg.BaseDN == "" {
		return nil, fmt.Errorf("%w: missing LDAP host or base DN", ErrInvalidConfig)
	}

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

func (p *LDAPProvider) GetAuthURL(_ context.Context, _ string) (string, error) {
	return "", ErrNotSupported
}

func (p *LDAPProvider) HandleCallback(_ context.Context, _ map[string]string) (*UserInfo, error) {
	return nil, ErrNotSupported
}

func (p *LDAPProvider) Authenticate(_ context.Context, username, password string) (*UserInfo, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("%w: username and password required", ErrAuthFailed)
	}

	conn, err := p.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer conn.Close()

	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return nil, fmt.Errorf("service account bind failed: %w", err)
		}
	}

	filter := strings.ReplaceAll(p.config.UserFilter, "{{username}}", ldapv3.EscapeFilter(username))
	searchReq := ldapv3.NewSearchRequest(
		p.config.BaseDN,
		ldapv3.ScopeWholeSubtree,
		ldapv3.NeverDerefAliases,
		1,
		30,
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

	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", ErrAuthFailed)
	}

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

	return ldapv3.DialURL(
		fmt.Sprintf("ldap://%s", addr),
		ldapv3.DialWithDialer(&net.Dialer{Timeout: ldapConnectTimeout}),
	)
}

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
