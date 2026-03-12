package sso

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidProtocol(t *testing.T) {
	assert.True(t, IsValidProtocol(ProtocolOIDC))
	assert.True(t, IsValidProtocol(ProtocolSAML))
	assert.True(t, IsValidProtocol(ProtocolLDAP))
	assert.False(t, IsValidProtocol("kerberos"))
	assert.False(t, IsValidProtocol(""))
	assert.False(t, IsValidProtocol("OIDC")) // case-sensitive
}

func TestConfig_TableName(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, "sso_configs", cfg.TableName())
}
