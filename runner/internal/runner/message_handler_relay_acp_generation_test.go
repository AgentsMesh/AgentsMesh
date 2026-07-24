package runner

import (
	"errors"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

func TestHandleAcpRelayCommandInGeneration(t *testing.T) {
	h := newTestHandler()
	pod := &Pod{PodKey: "generation"}
	mock := &mockPodIO{}
	relayClient := relay.NewMockClient("wss://relay.example.com")
	relayClient.SetConnected(true)
	podRelay := NewACPPodRelay(pod.PodKey, nil, nil, nil)

	h.handleAcpRelayCommandInGeneration(pod, RelayInboundContext{}, []byte(`{"type":"prompt"}`))
	if relayClient.CountSentByType(relay.MsgTypeAcpEvent) != 0 {
		t.Fatal("invalid generation context sent an event")
	}

	h.handleAcpRelayCommandInGeneration(pod, RelayInboundContext{
		IO: mock, Relay: podRelay, Client: relayClient,
	}, []byte(`{"type":"prompt","prompt":"hello"}`))
	if relayClient.CountSentByType(relay.MsgTypeAcpEvent) != 1 {
		t.Fatal("generation-owned prompt did not send its echo")
	}
}

func TestApplyAcpRelayCommandWithoutSessionAccess(t *testing.T) {
	h := newTestHandler()
	pod := &Pod{PodKey: "no-session"}
	io := &stubPodIO{}
	commands := []string{
		`{"type":"permission_response"}`,
		`{"type":"cancel"}`,
		`{"type":"interrupt"}`,
		`{"type":"set_permission_mode"}`,
		`{"type":"set_model"}`,
		`{"type":"control_request"}`,
	}
	for _, command := range commands {
		if outbound := h.applyAcpRelayCommand(pod, io, []byte(command)); len(outbound) != 0 {
			t.Fatalf("non-session command %s produced outbound events: %+v", command, outbound)
		}
	}
}

func TestApplyAcpRelayCommandFailurePaths(t *testing.T) {
	h := newTestHandler()
	pod := &Pod{PodKey: "failures"}

	t.Run("prompt", func(t *testing.T) {
		io := &mockPodIO{sendErr: errors.New("send")}
		outbound := h.applyAcpRelayCommand(pod, io, []byte(`{"type":"prompt","prompt":"x"}`))
		if len(outbound) != 1 || outbound[0].typ != "contentChunk" {
			t.Fatalf("prompt failure lost echo: %+v", outbound)
		}
	})

	t.Run("permission cancel and interrupt", func(t *testing.T) {
		io := &mockPodIO{
			permErr: errors.New("permission"), cancelErr: errors.New("cancel"), interruptErr: errors.New("interrupt"),
		}
		for _, command := range []string{
			`{"type":"permission_response","requestId":"id"}`,
			`{"type":"cancel"}`,
			`{"type":"interrupt"}`,
		} {
			h.applyAcpRelayCommand(pod, io, []byte(command))
		}
	})

	t.Run("configuration failures", func(t *testing.T) {
		io := &mockPodIO{permModeErr: errors.New("mode"), modelErr: errors.New("model")}
		for _, command := range []string{
			`{"type":"set_permission_mode","mode":"strict"}`,
			`{"type":"set_model","model":"model"}`,
		} {
			outbound := h.applyAcpRelayCommand(pod, io, []byte(command))
			if len(outbound) != 1 || outbound[0].typ != "configChangeFailed" {
				t.Fatalf("configuration failure %s = %+v", command, outbound)
			}
		}
	})

	t.Run("control error without response", func(t *testing.T) {
		io := &mockPodIO{controlErr: errors.New("control"), controlNil: true}
		outbound := h.applyAcpRelayCommand(pod, io, []byte(`{"type":"control_request","subtype":"status"}`))
		if len(outbound) != 0 {
			t.Fatalf("nil control response produced event: %+v", outbound)
		}
	})
}
