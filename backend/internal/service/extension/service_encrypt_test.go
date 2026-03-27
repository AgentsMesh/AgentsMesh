package extension

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
)

// ---------------------------------------------------------------------------
// Tests: encryptEnvVars
// ---------------------------------------------------------------------------

func TestEncryptEnvVars_CryptoNil_StoresPlainJSON(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	vars := map[string]string{"KEY": "value123"}
	data, err := svc.encryptEnvVars(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["KEY"] != "value123" {
		t.Errorf("expected plain 'value123', got %q", result["KEY"])
	}
}

func TestEncryptEnvVars_CryptoPresent_EncryptsEachValue(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	vars := map[string]string{
		"KEY_A": "value-a",
		"KEY_B": "value-b",
	}
	data, err := svc.encryptEnvVars(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Values should be encrypted (not plain text)
	if result["KEY_A"] == "value-a" {
		t.Error("KEY_A should be encrypted, not plain text")
	}
	if result["KEY_B"] == "value-b" {
		t.Error("KEY_B should be encrypted, not plain text")
	}

	// Verify decryption produces original values
	decA, err := enc.Decrypt(result["KEY_A"])
	if err != nil {
		t.Fatalf("failed to decrypt KEY_A: %v", err)
	}
	if decA != "value-a" {
		t.Errorf("expected 'value-a', got %q", decA)
	}

	decB, err := enc.Decrypt(result["KEY_B"])
	if err != nil {
		t.Fatalf("failed to decrypt KEY_B: %v", err)
	}
	if decB != "value-b" {
		t.Errorf("expected 'value-b', got %q", decB)
	}
}

// ---------------------------------------------------------------------------
// Tests: decryptServerEnvVars
// ---------------------------------------------------------------------------

func TestDecryptServerEnvVars_CryptoNil_ReturnsNil(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage(`{"KEY":"encrypted-val"}`),
	}
	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// EnvVars should remain unchanged (no-op)
	var envMap map[string]string
	if err := json.Unmarshal(server.EnvVars, &envMap); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if envMap["KEY"] != "encrypted-val" {
		t.Errorf("expected original value, got %q", envMap["KEY"])
	}
}

func TestDecryptServerEnvVars_EmptyEnvVars_ReturnsNil(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		EnvVars: nil,
	}
	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptServerEnvVars_EmptyJSONEnvVars_ReturnsNil(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage{},
	}
	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptServerEnvVars_DecryptFailureKeepsOriginal(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	// Store a value that is NOT valid encrypted text
	envJSON, _ := json.Marshal(map[string]string{"KEY": "plain-text-not-encrypted"})
	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage(envJSON),
	}

	err := svc.decryptServerEnvVars(server)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envMap map[string]string
	if err := json.Unmarshal(server.EnvVars, &envMap); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// When decryption fails, the original value is preserved
	if envMap["KEY"] != "plain-text-not-encrypted" {
		t.Errorf("expected original value preserved, got %q", envMap["KEY"])
	}
}

// ---------------------------------------------------------------------------
// Tests: decryptServerEnvVars (invalid JSON)
// ---------------------------------------------------------------------------

func TestDecryptServerEnvVars_InvalidJSON(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	server := &extension.InstalledMcpServer{
		EnvVars: json.RawMessage(`{invalid json`),
	}
	err := svc.decryptServerEnvVars(server)
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateSkill — repoID mismatch
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Tests: Encryption helpers (encryptCredential / decryptCredential)
// ---------------------------------------------------------------------------

func TestDecryptCredential_NoCrypto(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.DecryptCredential("some-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "some-value" {
		t.Errorf("expected 'some-value' returned as-is, got %q", result)
	}
}

func TestDecryptCredential_EmptyString(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	result, err := svc.DecryptCredential("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestDecryptCredential_Success(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	// Encrypt a value first
	encrypted, err := enc.Encrypt("my-secret-token")
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Decrypt via service
	result, err := svc.DecryptCredential(encrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "my-secret-token" {
		t.Errorf("expected 'my-secret-token', got %q", result)
	}
}

func TestEncryptCredential_NoCrypto(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	result, err := svc.encryptCredential("plain-value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "plain-value" {
		t.Errorf("expected 'plain-value' returned as-is (no crypto), got %q", result)
	}
}

func TestEncryptCredential_Success(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	encrypted, err := svc.encryptCredential("my-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Encrypted value should be different from plain text
	if encrypted == "my-secret" {
		t.Error("expected encrypted value to differ from plain text")
	}
	// Verify it can be decrypted back
	decrypted, err := enc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}
	if decrypted != "my-secret" {
		t.Errorf("expected 'my-secret', got %q", decrypted)
	}
}

// ---------------------------------------------------------------------------
// Tests: decryptCredential — additional coverage for decrypt-failure path
// ---------------------------------------------------------------------------

func TestDecryptCredential_CryptoPresent_DecryptFails_ReturnsOriginal(t *testing.T) {
	// When crypto is present but the value is not valid encrypted data,
	// decryptCredential should return the original value as-is (not an error).
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	// "not-encrypted-data" is not valid AES-GCM ciphertext
	result, err := svc.decryptCredential("not-encrypted-data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return the original value when decryption fails
	if result != "not-encrypted-data" {
		t.Errorf("expected original value 'not-encrypted-data', got %q", result)
	}
}

func TestDecryptCredential_CryptoPresent_DecryptFailsWithBase64Data_ReturnsOriginal(t *testing.T) {
	// Use a different key to encrypt, so decrypting with the service's key fails
	otherEnc := crypto.NewEncryptor("different-key-1234567890123456")
	encrypted, err := otherEnc.Encrypt("secret-value")
	if err != nil {
		t.Fatalf("failed to encrypt with other key: %v", err)
	}

	// Service uses a different key
	svcEnc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, svcEnc)

	// Decryption should fail (wrong key), return original value
	result, err := svc.decryptCredential(encrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != encrypted {
		t.Errorf("expected original encrypted value returned as-is, got different value")
	}
}

// ---------------------------------------------------------------------------
// Tests: encryptEnvVars — additional coverage
// ---------------------------------------------------------------------------

func TestEncryptEnvVars_EmptyMap_ReturnsEmptyJSONObject(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	data, err := svc.encryptEnvVars(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestEncryptEnvVars_NoCrypto_EmptyMap(t *testing.T) {
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, nil)

	data, err := svc.encryptEnvVars(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestEncryptEnvVars_MultipleKeys_AllEncrypted(t *testing.T) {
	enc := crypto.NewEncryptor("test-secret-key-1234567890123456")
	svc := newTestService(&svcMockRepo{}, &svcMockStorage{}, enc)

	vars := map[string]string{
		"DB_PASSWORD":  "secret1",
		"API_KEY":      "secret2",
		"WEBHOOK_TOKEN": "secret3",
	}
	data, err := svc.encryptEnvVars(vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// All keys should be present
	if len(result) != 3 {
		t.Errorf("expected 3 keys, got %d", len(result))
	}

	// Each value should be encrypted (different from plain text) and decryptable
	for k, plain := range vars {
		encrypted, ok := result[k]
		if !ok {
			t.Errorf("missing key %s", k)
			continue
		}
		if encrypted == plain {
			t.Errorf("key %s should be encrypted, not plain text", k)
		}
		decrypted, err := enc.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("failed to decrypt %s: %v", k, err)
		}
		if decrypted != plain {
			t.Errorf("expected %q for key %s, got %q", plain, k, decrypted)
		}
	}
}
