package agentkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubAdapter struct{ called bool }

func (a *stubAdapter) Adapt(data []byte) []byte {
	a.called = true
	return []byte("adapted")
}

func TestRegisterInputAdapter_And_Lookup(t *testing.T) {
	adapter := &stubAdapter{}
	RegisterInputAdapter("stub-agent", adapter)

	result := AdaptTerminalInput([]byte("hello"), "stub-agent")
	assert.Equal(t, []byte("adapted"), result)
	assert.True(t, adapter.called)
}

func TestRegisterInputAdapter_DuplicatePanics(t *testing.T) {
	RegisterInputAdapter("dup-adapter-test", &stubAdapter{})
	defer delete(inputAdapterRegistry, "dup-adapter-test")

	assert.Panics(t, func() {
		RegisterInputAdapter("dup-adapter-test", &stubAdapter{})
	})
}

func TestAdaptTerminalInput_UnregisteredPassesThrough(t *testing.T) {
	data := []byte("raw input")
	result := AdaptTerminalInput(data, "unknown-agent")
	assert.Equal(t, data, result)
}

func TestRegisterProcessNames_And_IsAgentProcess(t *testing.T) {
	RegisterProcessNames("test-proc-a", "test-proc-b")
	defer func() {
		delete(processNameSet, "test-proc-a")
		delete(processNameSet, "test-proc-b")
	}()

	assert.True(t, IsAgentProcess("test-proc-a"))
	assert.True(t, IsAgentProcess("test-proc-b"))
	assert.True(t, IsAgentProcess("node"))
	assert.False(t, IsAgentProcess("unknown-proc"))
}

func TestRegisterProcessNames_DuplicatePanics(t *testing.T) {
	RegisterProcessNames("dup-proc-test")
	defer delete(processNameSet, "dup-proc-test")

	assert.Panics(t, func() {
		RegisterProcessNames("dup-proc-test")
	})
}

func TestRegisterAgentHome_DuplicatePanics(t *testing.T) {
	original := append([]AgentHomeSpec{}, agentHomeSpecs...)
	defer func() { agentHomeSpecs = original }()

	RegisterAgentHome(AgentHomeSpec{EnvVar: "DUP_HOME_TEST", UserDirName: ".dup"})
	assert.Panics(t, func() {
		RegisterAgentHome(AgentHomeSpec{EnvVar: "DUP_HOME_TEST", UserDirName: ".dup2"})
	})
}

func TestRegisterAgentHome_And_Match(t *testing.T) {
	original := append([]AgentHomeSpec{}, agentHomeSpecs...)
	defer func() { agentHomeSpecs = original }()

	called := false
	RegisterAgentHome(AgentHomeSpec{
		EnvVar:      "TEST_AGENT_HOME",
		UserDirName: ".test-agent",
		MergeConfig: func(_, _ string) error { called = true; return nil },
	})

	spec, val := MatchAgentHome(map[string]string{"TEST_AGENT_HOME": "/tmp/home"})
	assert.NotNil(t, spec)
	assert.Equal(t, "/tmp/home", val)
	assert.Equal(t, ".test-agent", spec.UserDirName)

	_ = spec.MergeConfig("", "")
	assert.True(t, called)
}

func TestMatchAgentHome_NoMatch(t *testing.T) {
	spec, val := MatchAgentHome(map[string]string{"UNRELATED": "value"})
	assert.Nil(t, spec)
	assert.Empty(t, val)
}

func TestMatchAgentHome_EmptyValueSkipped(t *testing.T) {
	original := append([]AgentHomeSpec{}, agentHomeSpecs...)
	defer func() { agentHomeSpecs = original }()

	RegisterAgentHome(AgentHomeSpec{EnvVar: "EMPTY_TEST", UserDirName: ".empty"})

	spec, _ := MatchAgentHome(map[string]string{"EMPTY_TEST": ""})
	assert.Nil(t, spec)
}
