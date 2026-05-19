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

func parseSSHKey(privateKeyPEM, publicKeyStr string) (string, string, string, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPEM))
	if err != nil {
		slog.Error("failed to parse SSH private key", "error", err)
		return "", "", "", ErrInvalidSSHKey
	}

	pubKey := signer.PublicKey()
	publicKey := string(ssh.MarshalAuthorizedKey(pubKey))
	publicKey = strings.TrimSpace(publicKey)

	hash := sha256.Sum256(pubKey.Marshal())
	fingerprint := "SHA256:" + hex.EncodeToString(hash[:])

	return privateKeyPEM, publicKey, fingerprint, nil
}

func GenerateSSHKeyPair() (privateKey, publicKey string, err error) {
	pubKey, privKey, err := generateED25519Key()
	if err != nil {
		slog.Error("failed to generate ED25519 SSH key pair", "error", err)
		return "", "", err
	}
	return privKey, pubKey, nil
}

func generateED25519Key() (publicKey, privateKey string, err error) {
	seed := make([]byte, 32)
	if _, err := rand.Read(seed); err != nil {
		slog.Error("failed to generate random seed for SSH key", "error", err)
		return "", "", err
	}

	return "", "", errors.New("SSH key generation not implemented - please provide your own key")
}
