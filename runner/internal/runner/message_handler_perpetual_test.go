package runner

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func TestOnUpdatePodPerpetual_Enable(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	store.Put("pod-1", &Pod{PodKey: "pod-1", Perpetual: false})

	err := handler.OnUpdatePodPerpetual(&runnerv1.UpdatePodPerpetualCommand{
		PodKey:    "pod-1",
		Perpetual: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pod, ok := store.Get("pod-1")
	if !ok {
		t.Fatal("pod should still exist in store")
	}
	if !pod.Perpetual {
		t.Error("pod.Perpetual should be true after enable")
	}
}

func TestOnUpdatePodPerpetual_Disable(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	store.Put("pod-1", &Pod{PodKey: "pod-1", Perpetual: true})

	err := handler.OnUpdatePodPerpetual(&runnerv1.UpdatePodPerpetualCommand{
		PodKey:    "pod-1",
		Perpetual: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pod, ok := store.Get("pod-1")
	if !ok {
		t.Fatal("pod should still exist in store")
	}
	if pod.Perpetual {
		t.Error("pod.Perpetual should be false after disable")
	}
}

func TestOnUpdatePodPerpetual_PodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	err := handler.OnUpdatePodPerpetual(&runnerv1.UpdatePodPerpetualCommand{
		PodKey:    "nonexistent",
		Perpetual: true,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent pod")
	}
	if !contains(err.Error(), "pod not found") {
		t.Errorf("error = %v, want containing 'pod not found'", err)
	}
}
