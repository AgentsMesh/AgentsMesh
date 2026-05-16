package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReconcileServiceStatus_PromotesToStaleWhenConfigMissing(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist.yaml")

	if got := reconcileServiceStatus("Running", missing); got != "Stale" {
		t.Errorf("Running + missing config: got %q, want Stale", got)
	}
	if got := reconcileServiceStatus("Stopped", missing); got != "Stale" {
		t.Errorf("Stopped + missing config: got %q, want Stale", got)
	}
}

func TestReconcileServiceStatus_PassesThroughWhenConfigExists(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(cfg, []byte("node_id: foo\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if got := reconcileServiceStatus("Running", cfg); got != "Running" {
		t.Errorf("Running + config exists: got %q, want Running", got)
	}
	if got := reconcileServiceStatus("Stopped", cfg); got != "Stopped" {
		t.Errorf("Stopped + config exists: got %q, want Stopped", got)
	}
}

func TestReconcileServiceStatus_UnknownIsNeverPromoted(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "does-not-exist.yaml")
	if got := reconcileServiceStatus("Unknown", missing); got != "Unknown" {
		t.Errorf("Unknown should never be reclassified, got %q", got)
	}
}
