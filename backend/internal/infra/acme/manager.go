package acme

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"

	"github.com/anthropics/agentsmesh/backend/internal/infra/dns"
)

// Config holds ACME manager configuration
type Config struct {
	// ACME directory URL
	// Production: https://acme-v02.api.letsencrypt.org/directory
	// Staging: https://acme-staging-v02.api.letsencrypt.org/directory
	DirectoryURL string

	// Email for Let's Encrypt account registration
	Email string

	// Domain for wildcard certificate (e.g., "relay.agentsmesh.cn")
	// Will request certificate for "*.relay.agentsmesh.cn"
	Domain string

	// Storage directory for certificates and account data
	StorageDir string

	// DNS provider for DNS-01 challenge
	DNSProvider dns.Provider

	// Certificate renewal threshold (default: 30 days before expiry)
	RenewalDays int
}

// Manager handles ACME certificate management
type Manager struct {
	cfg    Config
	client *lego.Client
	user   *acmeUser

	// Current certificate
	cert      *Certificate
	certMu    sync.RWMutex

	logger *slog.Logger
}

// Certificate holds the certificate data
type Certificate struct {
	Domain      string    `json:"domain"`
	Certificate []byte    `json:"certificate"` // PEM encoded certificate chain
	PrivateKey  []byte    `json:"private_key"` // PEM encoded private key
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	IssuedAt    time.Time `json:"issued_at"`
}

// NewManager creates a new ACME manager
func NewManager(cfg Config) (*Manager, error) {
	if cfg.DirectoryURL == "" {
		cfg.DirectoryURL = lego.LEDirectoryProduction
	}
	if cfg.RenewalDays == 0 {
		cfg.RenewalDays = 30
	}
	if cfg.StorageDir == "" {
		cfg.StorageDir = "/var/lib/agentsmesh/acme"
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(cfg.StorageDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	m := &Manager{
		cfg:    cfg,
		logger: slog.With("component", "acme_manager"),
	}

	// Load or create user
	if err := m.loadOrCreateUser(); err != nil {
		return nil, fmt.Errorf("failed to load/create ACME user: %w", err)
	}

	// Create ACME client
	legoConfig := lego.NewConfig(m.user)
	legoConfig.CADirURL = cfg.DirectoryURL
	legoConfig.Certificate.KeyType = certcrypto.EC256

	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ACME client: %w", err)
	}

	// Set DNS-01 challenge provider
	dnsProvider := &dnsProviderAdapter{provider: cfg.DNSProvider, logger: m.logger}
	if err := client.Challenge.SetDNS01Provider(dnsProvider, dns01.AddRecursiveNameservers([]string{"8.8.8.8:53", "1.1.1.1:53"})); err != nil {
		return nil, fmt.Errorf("failed to set DNS provider: %w", err)
	}

	m.client = client

	// Register user if not already registered
	if m.user.Registration == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("failed to register ACME user: %w", err)
		}
		m.user.Registration = reg
		if err := m.saveUser(); err != nil {
			return nil, fmt.Errorf("failed to save user registration: %w", err)
		}
		m.logger.Info("ACME user registered", "email", cfg.Email)
	}

	// Load existing certificate if available
	if err := m.loadCertificate(); err != nil {
		m.logger.Warn("No existing certificate found", "error", err)
	}

	m.logger.Info("ACME manager initialized",
		"directory", cfg.DirectoryURL,
		"domain", cfg.Domain,
		"email", cfg.Email)

	return m, nil
}
