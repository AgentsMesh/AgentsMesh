package agent

import (
	"testing"
	"time"
)

// --- Test EncryptedCredentials ---

func TestEncryptedCredentialsScanNil(t *testing.T) {
	var ec EncryptedCredentials
	err := ec.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ec != nil {
		t.Error("expected nil EncryptedCredentials")
	}
}

func TestEncryptedCredentialsScanValid(t *testing.T) {
	var ec EncryptedCredentials
	err := ec.Scan([]byte(`{"api_key":"encrypted_value"}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ec["api_key"] != "encrypted_value" {
		t.Errorf("expected 'encrypted_value', got %s", ec["api_key"])
	}
}

func TestEncryptedCredentialsScanInvalidType(t *testing.T) {
	var ec EncryptedCredentials
	err := ec.Scan("not bytes")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestEncryptedCredentialsValueNil(t *testing.T) {
	var ec EncryptedCredentials
	val, err := ec.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value")
	}
}

func TestEncryptedCredentialsValueValid(t *testing.T) {
	ec := EncryptedCredentials{"api_key": "encrypted"}
	val, err := ec.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}

// --- Test Agent ---

func TestAgentTableName(t *testing.T) {
	at := Agent{}
	if at.TableName() != "agents" {
		t.Errorf("expected 'agents', got %s", at.TableName())
	}
}

func TestAgentStruct(t *testing.T) {
	now := time.Now()
	desc := "Test agent"
	args := "--verbose"

	at := Agent{
		Slug:          "test-agent",
		Name:          "Test Agent",
		Description:   &desc,
		LaunchCommand: "test-cli",
		DefaultArgs:   &args,
		IsBuiltin:     true,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if at.Slug != "test-agent" {
		t.Errorf("expected slug test-agent, got %s", at.Slug)
	}
	if at.Slug != "test-agent" {
		t.Errorf("expected Slug 'test-agent', got %s", at.Slug)
	}
	if at.LaunchCommand != "test-cli" {
		t.Errorf("expected LaunchCommand 'test-cli', got %s", at.LaunchCommand)
	}
}

// --- Test CustomAgent ---

func TestCustomAgentTableName(t *testing.T) {
	cat := CustomAgent{}
	if cat.TableName() != "custom_agents" {
		t.Errorf("expected 'custom_agents', got %s", cat.TableName())
	}
}

func TestCustomAgentStruct(t *testing.T) {
	now := time.Now()
	desc := "Custom agent"

	cat := CustomAgent{
		OrganizationID: 100,
		Slug:           "custom-agent",
		Name:           "Custom Agent",
		Description:    &desc,
		LaunchCommand:  "custom-cli",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if cat.Slug != "custom-agent" {
		t.Errorf("expected slug test-agent, got %s", cat.Slug)
	}
	if cat.OrganizationID != 100 {
		t.Errorf("expected OrganizationID 100, got %d", cat.OrganizationID)
	}
	if cat.Slug != "custom-agent" {
		t.Errorf("expected Slug 'custom-agent', got %s", cat.Slug)
	}
}

// --- Test SupportsMode ---

func TestAgentTypeSupportsMode(t *testing.T) {
	tests := []struct {
		name           string
		supportedModes string
		mode           string
		expected       bool
	}{
		// Agent with multiple supported modes
		{"pty supported in pty,acp", "pty,acp", "pty", true},
		{"acp supported in pty,acp", "pty,acp", "acp", true},
		{"unknown not supported in pty,acp", "pty,acp", "unknown", false},

		// Default SupportedModes (DB default is "pty")
		{"pty supported when default pty", "pty", "pty", true},
		{"acp not supported when default pty", "pty", "acp", false},

		// Agent that only supports acp
		{"acp supported in acp-only", "acp", "acp", true},
		{"pty not supported in acp-only", "acp", "pty", false},

		// Whitespace trimming
		{"trims spaces around modes", "pty, acp", "acp", true},
		{"trims spaces for exact match", " pty , acp ", "pty", true},

		// Empty mode query
		{"empty mode not matched", "pty,acp", "", false},

		// Empty SupportedModes field
		{"empty supported_modes matches nothing", "", "pty", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at := &Agent{SupportedModes: tt.supportedModes}
			got := at.SupportsMode(tt.mode)
			if got != tt.expected {
				t.Errorf("SupportsMode(%q) = %v, want %v", tt.mode, got, tt.expected)
			}
		})
	}
}

// --- Benchmark Tests ---

func BenchmarkAgentTableName(b *testing.B) {
	at := Agent{}
	for i := 0; i < b.N; i++ {
		at.TableName()
	}
}
