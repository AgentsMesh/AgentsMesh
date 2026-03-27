package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"
)

// loadPrivateKey loads an RSA private key from a PEM file
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Try PKCS8 first
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}

	// Try PKCS1
	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return rsaKey, nil
}

// signLicense signs the license data using RSA-SHA256
func signLicense(license *LicenseData, privateKey *rsa.PrivateKey) (string, error) {
	// Build the data to sign (everything except signature)
	dataToSign := struct {
		LicenseKey       string        `json:"license_key"`
		OrganizationName string        `json:"organization_name"`
		ContactEmail     string        `json:"contact_email"`
		Plan             string        `json:"plan"`
		Limits           LicenseLimits `json:"limits"`
		Features         []string      `json:"features,omitempty"`
		IssuedAt         time.Time     `json:"issued_at"`
		ExpiresAt        time.Time     `json:"expires_at"`
	}{
		LicenseKey:       license.LicenseKey,
		OrganizationName: license.OrganizationName,
		ContactEmail:     license.ContactEmail,
		Plan:             license.Plan,
		Limits:           license.Limits,
		Features:         license.Features,
		IssuedAt:         license.IssuedAt,
		ExpiresAt:        license.ExpiresAt,
	}

	jsonData, err := json.Marshal(dataToSign)
	if err != nil {
		return "", fmt.Errorf("failed to marshal license data: %w", err)
	}

	// Hash the data
	hash := sha256.Sum256(jsonData)

	// Sign with RSA PKCS1v15
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// generateKeyPair generates a new RSA key pair
func generateKeyPair(outputDir string) error {
	fmt.Println("Generating RSA 4096-bit key pair...")

	// Generate key
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Encode private key
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Encode public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write private key
	privateKeyPath := fmt.Sprintf("%s/license_private.pem", strings.TrimSuffix(outputDir, "/"))
	if err := os.WriteFile(privateKeyPath, privateKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Write public key
	publicKeyPath := fmt.Sprintf("%s/license_public.pem", strings.TrimSuffix(outputDir, "/"))
	if err := os.WriteFile(publicKeyPath, publicKeyPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("Key pair generated successfully!\n")
	fmt.Printf("  Private key: %s (keep this secure, use for signing licenses)\n", privateKeyPath)
	fmt.Printf("  Public key: %s (deploy to OnPremise installations for verification)\n", publicKeyPath)

	return nil
}
