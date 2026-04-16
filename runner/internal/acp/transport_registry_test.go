package acp

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterCommandMapping_And_Lookup(t *testing.T) {
	RegisterCommandMapping("test-agent-cmd", "test-transport-cmd")
	defer func() {
		registryMu.Lock()
		delete(commandMap, "test-agent-cmd")
		registryMu.Unlock()
	}()

	assert.Equal(t, "test-transport-cmd", TransportTypeForCommand("test-agent-cmd"))
}

func TestTransportTypeForCommand_UnknownFallsBackToACP(t *testing.T) {
	assert.Equal(t, TransportTypeACP, TransportTypeForCommand("nonexistent-agent"))
}

func TestRegisterCommandMapping_DuplicatePanics(t *testing.T) {
	RegisterCommandMapping("dup-cmd-agent", "type-a")
	defer func() {
		registryMu.Lock()
		delete(commandMap, "dup-cmd-agent")
		registryMu.Unlock()
	}()

	assert.Panics(t, func() {
		RegisterCommandMapping("dup-cmd-agent", "type-b")
	})
}

func TestRegisterTransport_DuplicatePanics(t *testing.T) {
	RegisterTransport("dup-transport", func(_ EventCallbacks, _ *slog.Logger) Transport { return nil })
	defer func() {
		registryMu.Lock()
		delete(registry, "dup-transport")
		registryMu.Unlock()
	}()

	assert.Panics(t, func() {
		RegisterTransport("dup-transport", nil)
	})
}

func TestRegisterAgent_RegistersBoth(t *testing.T) {
	RegisterAgent("agent-both-cmd", "agent-both-type", func(_ EventCallbacks, _ *slog.Logger) Transport { return nil })
	defer func() {
		registryMu.Lock()
		delete(commandMap, "agent-both-cmd")
		delete(registry, "agent-both-type")
		registryMu.Unlock()
	}()

	assert.Equal(t, "agent-both-type", TransportTypeForCommand("agent-both-cmd"))
}

func TestNewTransport_UnknownFallsBackToACP(t *testing.T) {
	tr := NewTransport("totally-unknown-transport", EventCallbacks{}, slog.Default())
	assert.NotNil(t, tr)
}
