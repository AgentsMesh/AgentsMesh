package acp

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestParseAgentsmeshExtensions(t *testing.T) {
	t.Run("loopal advertises controlRequest + permissionModes", func(t *testing.T) {
		raw := json.RawMessage(`{"agentsmeshExtensions":{"controlRequest":true,"permissionModes":["bypass","ask_dangerous","ask_any_write"]}}`)
		ctrl, modes := parseAgentsmeshExtensions(raw)
		if !ctrl {
			t.Error("controlRequest = false, want true")
		}
		want := []string{"bypass", "ask_dangerous", "ask_any_write"}
		if !slices.Equal(modes, want) {
			t.Errorf("permissionModes = %v, want %v", modes, want)
		}
	})

	t.Run("no extension block leaves both zero", func(t *testing.T) {
		ctrl, modes := parseAgentsmeshExtensions(json.RawMessage(`{"protocolVersion":1}`))
		if ctrl || modes != nil {
			t.Errorf("got (%v, %v), want (false, nil)", ctrl, modes)
		}
	})

	t.Run("controlRequest only leaves modes nil", func(t *testing.T) {
		ctrl, modes := parseAgentsmeshExtensions(json.RawMessage(`{"agentsmeshExtensions":{"controlRequest":true}}`))
		if !ctrl || modes != nil {
			t.Errorf("got (%v, %v), want (true, nil)", ctrl, modes)
		}
	})

	t.Run("invalid json leaves both zero", func(t *testing.T) {
		ctrl, modes := parseAgentsmeshExtensions(json.RawMessage(`not json`))
		if ctrl || modes != nil {
			t.Errorf("got (%v, %v), want (false, nil)", ctrl, modes)
		}
	})
}
