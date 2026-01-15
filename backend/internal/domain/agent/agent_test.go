package agent

import (
	"testing"
	"time"
)

// --- Test CredentialSchema ---

func TestCredentialSchemaScanNil(t *testing.T) {
	var cs CredentialSchema
	err := cs.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cs != nil {
		t.Error("expected nil CredentialSchema")
	}
}

func TestCredentialSchemaScanValid(t *testing.T) {
	var cs CredentialSchema
	err := cs.Scan([]byte(`[{"name":"api_key","type":"secret","env_var":"API_KEY","required":true}]`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(cs) != 1 {
		t.Errorf("expected 1 field, got %d", len(cs))
	}
	if cs[0].Name != "api_key" {
		t.Errorf("expected Name 'api_key', got %s", cs[0].Name)
	}
	if cs[0].Type != "secret" {
		t.Errorf("expected Type 'secret', got %s", cs[0].Type)
	}
	if cs[0].EnvVar != "API_KEY" {
		t.Errorf("expected EnvVar 'API_KEY', got %s", cs[0].EnvVar)
	}
	if !cs[0].Required {
		t.Error("expected Required true")
	}
}

func TestCredentialSchemaScanInvalidType(t *testing.T) {
	var cs CredentialSchema
	err := cs.Scan("not bytes")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestCredentialSchemaScanInvalidJSON(t *testing.T) {
	var cs CredentialSchema
	err := cs.Scan([]byte(`invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCredentialSchemaValueNil(t *testing.T) {
	var cs CredentialSchema
	val, err := cs.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value")
	}
}

func TestCredentialSchemaValueValid(t *testing.T) {
	cs := CredentialSchema{
		{Name: "api_key", Type: "secret", EnvVar: "API_KEY", Required: true},
	}
	val, err := cs.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}

// --- Test StatusDetection ---

func TestStatusDetectionScanNil(t *testing.T) {
	var sd StatusDetection
	err := sd.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if sd != nil {
		t.Error("expected nil StatusDetection")
	}
}

func TestStatusDetectionScanValid(t *testing.T) {
	var sd StatusDetection
	err := sd.Scan([]byte(`{"pattern":"working","type":"regex"}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if sd["pattern"] != "working" {
		t.Errorf("expected pattern 'working', got %v", sd["pattern"])
	}
}

func TestStatusDetectionScanInvalidType(t *testing.T) {
	var sd StatusDetection
	err := sd.Scan("not bytes")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestStatusDetectionValueNil(t *testing.T) {
	var sd StatusDetection
	val, err := sd.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value")
	}
}

func TestStatusDetectionValueValid(t *testing.T) {
	sd := StatusDetection{"pattern": "working"}
	val, err := sd.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}

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

// --- Test AgentType ---

func TestAgentTypeTableName(t *testing.T) {
	at := AgentType{}
	if at.TableName() != "agent_types" {
		t.Errorf("expected 'agent_types', got %s", at.TableName())
	}
}

func TestAgentTypeStruct(t *testing.T) {
	now := time.Now()
	desc := "Test agent"
	args := "--verbose"

	at := AgentType{
		ID:               1,
		Slug:             "test-agent",
		Name:             "Test Agent",
		Description:      &desc,
		LaunchCommand:    "test-cli",
		DefaultArgs:      &args,
		CredentialSchema: CredentialSchema{{Name: "api_key", Type: "secret"}},
		IsBuiltin:        true,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if at.ID != 1 {
		t.Errorf("expected ID 1, got %d", at.ID)
	}
	if at.Slug != "test-agent" {
		t.Errorf("expected Slug 'test-agent', got %s", at.Slug)
	}
	if at.LaunchCommand != "test-cli" {
		t.Errorf("expected LaunchCommand 'test-cli', got %s", at.LaunchCommand)
	}
}


// --- Test UserAgentCredential ---

func TestUserAgentCredentialTableName(t *testing.T) {
	uac := UserAgentCredential{}
	if uac.TableName() != "user_agent_credentials" {
		t.Errorf("expected 'user_agent_credentials', got %s", uac.TableName())
	}
}

func TestUserAgentCredentialStruct(t *testing.T) {
	now := time.Now()

	uac := UserAgentCredential{
		ID:                   1,
		UserID:               50,
		AgentTypeID:          10,
		CredentialsEncrypted: EncryptedCredentials{"key": "value"},
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if uac.ID != 1 {
		t.Errorf("expected ID 1, got %d", uac.ID)
	}
	if uac.UserID != 50 {
		t.Errorf("expected UserID 50, got %d", uac.UserID)
	}
}

// --- Test CustomAgentType ---

func TestCustomAgentTypeTableName(t *testing.T) {
	cat := CustomAgentType{}
	if cat.TableName() != "custom_agent_types" {
		t.Errorf("expected 'custom_agent_types', got %s", cat.TableName())
	}
}

func TestCustomAgentTypeStruct(t *testing.T) {
	now := time.Now()
	desc := "Custom agent"

	cat := CustomAgentType{
		ID:             1,
		OrganizationID: 100,
		Slug:           "custom-agent",
		Name:           "Custom Agent",
		Description:    &desc,
		LaunchCommand:  "custom-cli",
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if cat.ID != 1 {
		t.Errorf("expected ID 1, got %d", cat.ID)
	}
	if cat.OrganizationID != 100 {
		t.Errorf("expected OrganizationID 100, got %d", cat.OrganizationID)
	}
	if cat.Slug != "custom-agent" {
		t.Errorf("expected Slug 'custom-agent', got %s", cat.Slug)
	}
}

// --- Benchmark Tests ---

func BenchmarkCredentialSchemaScan(b *testing.B) {
	data := []byte(`[{"name":"api_key","type":"secret","env_var":"API_KEY","required":true}]`)
	for i := 0; i < b.N; i++ {
		var cs CredentialSchema
		cs.Scan(data)
	}
}

func BenchmarkCredentialSchemaValue(b *testing.B) {
	cs := CredentialSchema{{Name: "api_key", Type: "secret"}}
	for i := 0; i < b.N; i++ {
		cs.Value()
	}
}

func BenchmarkStatusDetectionScan(b *testing.B) {
	data := []byte(`{"pattern":"working","type":"regex"}`)
	for i := 0; i < b.N; i++ {
		var sd StatusDetection
		sd.Scan(data)
	}
}

func BenchmarkAgentTypeTableName(b *testing.B) {
	at := AgentType{}
	for i := 0; i < b.N; i++ {
		at.TableName()
	}
}
