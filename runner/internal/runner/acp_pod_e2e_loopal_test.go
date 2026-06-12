//go:build integration

package runner

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// loopalSessionID matches mockSessionID used by runMockACPAgent.
const loopalSessionID = "mock-session-001"

//go:embed testdata/loopal_panel_signals.json
var loopalGoldenRaw []byte

type loopalSignal struct {
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// loopalGoldenSignals reads the canonical `_loopal/*` wire fixture shared with
// Loopal (see testdata/loopal_panel_signals.json header). The mock emitter and
// the e2e assertion both read this one definition, so every field is
// single-sourced — drift can only enter through the fixture, which Loopal's
// translate_event_test guards against panel.rs changes.
func loopalGoldenSignals() []loopalSignal {
	var fixture struct {
		Signals []loopalSignal `json:"signals"`
	}
	_ = json.Unmarshal(loopalGoldenRaw, &fixture)
	return fixture.Signals
}

// emitLoopalSignals makes the mock agent speak Loopal's `_loopal/*` dialect by
// replaying the shared golden fixture verbatim — no hand-written field shapes,
// so the mock cannot silently diverge from what the fixture (and Loopal) say.
func emitLoopalSignals(w *acp.Writer) {
	for _, sig := range loopalGoldenSignals() {
		_ = w.WriteNotification("_loopal/"+sig.Kind, map[string]any{
			"sessionId": loopalSessionID,
			"data":      sig.Data,
		})
	}
}

// handleMockLoopalControl accepts any loopal.* control subtype and returns ok,
// so the runner's control_request round-trip is exercised. Subtype validation
// is Loopal's job (tested in the loopal-acp repo); the mock proves transport.
func handleMockLoopalControl(w *acp.Writer, id int64, raw json.RawMessage) error {
	var req struct {
		Subtype string `json:"subtype"`
	}
	_ = json.Unmarshal(raw, &req)
	if strings.HasPrefix(req.Subtype, "loopal.") {
		return w.WriteResponse(id, map[string]any{"ok": true, "subtype": req.Subtype}, nil)
	}
	return w.WriteResponse(id, nil, &acp.JSONRPCError{
		Code: acp.ErrCodeMethodNotFound, Message: "unknown subtype: " + req.Subtype,
	})
}

// TestACPPod_Loopal_Signals_Integration drives a mock Loopal agent through the
// real ACPClient and asserts every golden `_loopal/*` panel signal lands on
// OnLoopalExt with byte-equal data (allowlist forwards all kinds AND drops no
// field) and accumulates into the resubscribe snapshot.
func TestACPPod_Loopal_Signals_Integration(t *testing.T) {
	golden := loopalGoldenSignals()
	if len(golden) == 0 {
		t.Fatal("golden fixture empty — embed/parse failed")
	}

	var mu sync.Mutex
	got := map[string]json.RawMessage{}

	acpClient := acp.NewClient(acp.ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     append(mockAgentEnv(), "ACP_MOCK_LOOPAL=1"),
		WorkDir: t.TempDir(),
		Logger:  slog.Default(),
		Callbacks: acp.EventCallbacks{
			OnLoopalExt: func(_, kind string, data json.RawMessage) {
				mu.Lock()
				got[kind] = data
				mu.Unlock()
			},
		},
	})
	if err := acpClient.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer acpClient.Stop()
	if err := acpClient.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := acpClient.SendPrompt("go"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n >= len(golden) {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timeout: got %d/%d kinds: %v", n, len(golden), mapKeys(got))
			mu.Unlock()
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	for _, sig := range golden {
		data, ok := got[sig.Kind]
		if !ok {
			t.Errorf("missing _loopal/%s on OnLoopalExt", sig.Kind)
			continue
		}
		if !jsonEqual(data, sig.Data) {
			t.Errorf("_loopal/%s data drift:\n got=%s\nwant=%s", sig.Kind, data, sig.Data)
		}
	}

	if snap := acpClient.LoopalSnapshot(); snap == nil {
		t.Error("LoopalSnapshot() nil after _loopal/* signals")
	}
}

// TestACPPod_Loopal_Control_Integration verifies the reverse control path:
// ACPClient.SendControlRequest with a loopal.* subtype reaches the agent and
// returns ok (capability advertised in initialize).
func TestACPPod_Loopal_Control_Integration(t *testing.T) {
	acpClient := acp.NewClient(acp.ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     append(mockAgentEnv(), "ACP_MOCK_LOOPAL=1"),
		WorkDir: t.TempDir(),
		Logger:  slog.Default(),
	})
	if err := acpClient.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer acpClient.Stop()
	if err := acpClient.NewSession(nil); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	resp, err := acpClient.SendControlRequest("loopal.bgTaskKill", map[string]any{"id": "bg1"})
	if err != nil {
		t.Fatalf("SendControlRequest(loopal.bgTaskKill): %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("control response = %v, want ok=true", resp)
	}
}

func jsonEqual(a, b json.RawMessage) bool {
	var ia, ib any
	if json.Unmarshal(a, &ia) != nil || json.Unmarshal(b, &ib) != nil {
		return false
	}
	return reflect.DeepEqual(ia, ib)
}

func mapKeys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
