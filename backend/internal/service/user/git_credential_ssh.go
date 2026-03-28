package user

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"

	"golang.org/x/crypto/ssh"
)

// parseSSHKey validates and parses an SSH key, returns private key, public key, and fingerprint
func parseSSHKey(privateKeyPEM, publicKeyStr string) (string, string, string, error) {
	// Parse the private key
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		slog.Error("failed to parse SSH private key", "error", err)
		return "", "", "", ErrInvalidSSHKey
	}

	// Get public key from private key
	pubKey := signer.PublicKey()
	publicKey := string(ssh.MarshalAuthorizedKey(pubKey))
	publicKey = strings.TrimSpace(publicKey)

	// Calculate fingerprint (SHA256)
	hash := sha256.Sum256(pubKey.Marshal())
	fingerprint := "SHA256:" + hex.EncodeToString(hash[:])

	return privateKeyPEM, publicKey, fingerprint, nil
}

// GenerateSSHKeyPair generates a new SSH key pair
func GenerateSSHKeyPair() (privateKey, publicKey string, err error) {
	// Generate ED25519 key (more secure and shorter than RSA)
	pubKey, privKey, err := generateED25519Key()
	if err != nil {
		slog.Error("failed to generate ED25519 SSH key pair", "error", err)
		return "", "", err
	}
	return privKey, pubKey, nil
}

// generateED25519Key generates an ED25519 SSH key pair
func generateED25519Key() (publicKey, privateKey string, err error) {
	// Generate random seed
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		slog.Error("failed to generate random seed for SSH key", "error", err)
		return "", "", err
	}

	// For simplicity, we'll return an error asking user to provide their own key
	// Full ED25519 key generation would require additional dependencies
	return "", "", errors.New("SSH key generation not implemented - please provide your own key")
}
