package runner

import (
	"testing"
	"time"
)

// --- Test Runner Status Constants ---

func TestRunnerStatusConstants(t *testing.T) {
	if RunnerStatusOnline != "online" {
		t.Errorf("expected 'online', got %s", RunnerStatusOnline)
	}
	if RunnerStatusOffline != "offline" {
		t.Errorf("expected 'offline', got %s", RunnerStatusOffline)
	}
	if RunnerStatusBusy != "busy" {
		t.Errorf("expected 'busy', got %s", RunnerStatusBusy)
	}
}

// --- Test Runner ---

func TestRunnerTableName(t *testing.T) {
	runner := Runner{}
	if runner.TableName() != "runners" {
		t.Errorf("expected TableName 'runners', got %s", runner.TableName())
	}
}

func TestRunnerIsOnline(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"online status", RunnerStatusOnline, true},
		{"offline status", RunnerStatusOffline, false},
		{"busy status", RunnerStatusBusy, false},
		{"empty status", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{Status: tt.status}
			if r.IsOnline() != tt.expected {
				t.Errorf("expected IsOnline() = %v, got %v", tt.expected, r.IsOnline())
			}
		})
	}
}

func TestRunnerCanAcceptPod(t *testing.T) {
	tests := []struct {
		name        string
		isEnabled   bool
		status      string
		currentPods int
		maxPods     int
		expected    bool
	}{
		{
			name:        "can accept - all conditions met",
			isEnabled:   true,
			status:      RunnerStatusOnline,
			currentPods: 2,
			maxPods:     5,
			expected:    true,
		},
		{
			name:        "cannot accept - disabled",
			isEnabled:   false,
			status:      RunnerStatusOnline,
			currentPods: 2,
			maxPods:     5,
			expected:    false,
		},
		{
			name:        "cannot accept - offline",
			isEnabled:   true,
			status:      RunnerStatusOffline,
			currentPods: 2,
			maxPods:     5,
			expected:    false,
		},
		{
			name:        "cannot accept - at capacity",
			isEnabled:   true,
			status:      RunnerStatusOnline,
			currentPods: 5,
			maxPods:     5,
			expected:    false,
		},
		{
			name:        "cannot accept - over capacity",
			isEnabled:   true,
			status:      RunnerStatusOnline,
			currentPods: 6,
			maxPods:     5,
			expected:    false,
		},
		{
			name:        "can accept - one slot left",
			isEnabled:   true,
			status:      RunnerStatusOnline,
			currentPods: 4,
			maxPods:     5,
			expected:    true,
		},
		{
			name:        "can accept - zero pods",
			isEnabled:   true,
			status:      RunnerStatusOnline,
			currentPods: 0,
			maxPods:     5,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				IsEnabled:         tt.isEnabled,
				Status:            tt.status,
				CurrentPods:       tt.currentPods,
				MaxConcurrentPods: tt.maxPods,
			}
			if r.CanAcceptPod() != tt.expected {
				t.Errorf("expected CanAcceptPod() = %v, got %v", tt.expected, r.CanAcceptPod())
			}
		})
	}
}

func TestRunnerStruct(t *testing.T) {
	now := time.Now()
	version := "1.0.0"

	r := Runner{
		ID:                1,
		OrganizationID:    100,
		NodeID:            "node-001",
		Description:       "Test runner",
		Status:            RunnerStatusOnline,
		LastHeartbeat:     &now,
		CurrentPods:       3,
		MaxConcurrentPods: 10,
		RunnerVersion:     &version,
		IsEnabled:         true,
		HostInfo:          HostInfo{"os": "linux"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if r.ID != 1 {
		t.Errorf("expected ID 1, got %d", r.ID)
	}
	if r.NodeID != "node-001" {
		t.Errorf("expected NodeID 'node-001', got %s", r.NodeID)
	}
	if *r.RunnerVersion != "1.0.0" {
		t.Errorf("expected RunnerVersion '1.0.0', got %s", *r.RunnerVersion)
	}
	if r.HostInfo["os"] != "linux" {
		t.Errorf("expected HostInfo os 'linux', got %v", r.HostInfo["os"])
	}
}

// --- Benchmark Tests ---

func BenchmarkRunnerIsOnline(b *testing.B) {
	r := &Runner{Status: RunnerStatusOnline}
	for i := 0; i < b.N; i++ {
		r.IsOnline()
	}
}

func BenchmarkRunnerCanAcceptPod(b *testing.B) {
	r := &Runner{
		IsEnabled:         true,
		Status:            RunnerStatusOnline,
		CurrentPods:       2,
		MaxConcurrentPods: 5,
	}
	for i := 0; i < b.N; i++ {
		r.CanAcceptPod()
	}
}
