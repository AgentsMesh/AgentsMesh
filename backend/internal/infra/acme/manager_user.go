package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/registration"
)

// acmeUser implements registration.User interface
type acmeUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	Key          crypto.PrivateKey      `json:"-"`
	KeyPEM       []byte                 `json:"key_pem"`
}

func (u *acmeUser) GetEmail() string {
	return u.Email
}

func (u *acmeUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *acmeUser) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

// loadOrCreateUser loads existing user or creates a new one
func (m *Manager) loadOrCreateUser() error {
	userPath := filepath.Join(m.cfg.StorageDir, "user.json")

	data, err := os.ReadFile(userPath)
	if err == nil {
		// Load existing user
		var user acmeUser
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("failed to unmarshal user: %w", err)
		}

		// Decode private key
		block, _ := pem.Decode(user.KeyPEM)
		if block == nil {
			return fmt.Errorf("failed to decode user key PEM")
		}

		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse user key: %w", err)
		}
		user.Key = key

		m.user = &user
		m.logger.Info("Loaded existing ACME user", "email", user.Email)
		return nil
	}

	// Create new user
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate user key: %w", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal user key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	m.user = &acmeUser{
		Email:  m.cfg.Email,
		Key:    privateKey,
		KeyPEM: keyPEM,
	}

	m.logger.Info("Created new ACME user", "email", m.cfg.Email)
	return nil
}

// saveUser saves user data to disk
func (m *Manager) saveUser() error {
	userPath := filepath.Join(m.cfg.StorageDir, "user.json")

	data, err := json.MarshalIndent(m.user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	if err := os.WriteFile(userPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write user file: %w", err)
	}

	return nil
}
