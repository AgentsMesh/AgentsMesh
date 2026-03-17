package channel

import (
	"testing"
	"time"

	"github.com/lib/pq"
)

// --- Test Binding Status Constants ---

func TestBindingStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{BindingStatusPending, "pending"},
		{BindingStatusActive, "active"},
		{BindingStatusRejected, "rejected"},
		{BindingStatusInactive, "inactive"},
		{BindingStatusExpired, "expired"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test Binding Scope Constants ---

func TestBindingScopeConstants(t *testing.T) {
	if BindingScopePodRead != "pod:read" {
		t.Errorf("expected 'pod:read', got %s", BindingScopePodRead)
	}
	if BindingScopePodWrite != "pod:write" {
		t.Errorf("expected 'pod:write', got %s", BindingScopePodWrite)
	}
}

func TestValidBindingScopes(t *testing.T) {
	if !ValidBindingScopes[BindingScopePodRead] {
		t.Error("expected pod:read to be valid")
	}
	if !ValidBindingScopes[BindingScopePodWrite] {
		t.Error("expected pod:write to be valid")
	}
	if ValidBindingScopes["invalid:scope"] {
		t.Error("expected invalid:scope to be invalid")
	}
}

// --- Test Binding Policy Constants ---

func TestBindingPolicyConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{BindingPolicySameUserAuto, "same_user_auto"},
		{BindingPolicySameProjectAuto, "same_project_auto"},
		{BindingPolicyExplicitOnly, "explicit_only"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test PodBinding ---

func TestPodBindingTableName(t *testing.T) {
	pb := PodBinding{}
	if pb.TableName() != "pod_bindings" {
		t.Errorf("expected 'pod_bindings', got %s", pb.TableName())
	}
}

func TestPodBindingHasScope(t *testing.T) {
	pb := &PodBinding{
		GrantedScopes: pq.StringArray{BindingScopePodRead, BindingScopePodWrite},
	}

	if !pb.HasScope(BindingScopePodRead) {
		t.Error("expected HasScope(pod:read) = true")
	}
	if !pb.HasScope(BindingScopePodWrite) {
		t.Error("expected HasScope(pod:write) = true")
	}
	if pb.HasScope("invalid:scope") {
		t.Error("expected HasScope(invalid:scope) = false")
	}
}

func TestPodBindingHasPendingScope(t *testing.T) {
	pb := &PodBinding{
		PendingScopes: pq.StringArray{BindingScopePodWrite},
	}

	if !pb.HasPendingScope(BindingScopePodWrite) {
		t.Error("expected HasPendingScope(pod:write) = true")
	}
	if pb.HasPendingScope(BindingScopePodRead) {
		t.Error("expected HasPendingScope(pod:read) = false")
	}
}

func TestPodBindingIsActive(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{BindingStatusActive, true},
		{BindingStatusPending, false},
		{BindingStatusRejected, false},
		{BindingStatusInactive, false},
	}

	for _, tt := range tests {
		pb := &PodBinding{Status: tt.status}
		if pb.IsActive() != tt.expected {
			t.Errorf("status %s: expected IsActive() = %v", tt.status, tt.expected)
		}
	}
}

func TestPodBindingIsPending(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{BindingStatusPending, true},
		{BindingStatusActive, false},
		{BindingStatusRejected, false},
	}

	for _, tt := range tests {
		pb := &PodBinding{Status: tt.status}
		if pb.IsPending() != tt.expected {
			t.Errorf("status %s: expected IsPending() = %v", tt.status, tt.expected)
		}
	}
}

func TestPodBindingStruct(t *testing.T) {
	now := time.Now()
	reason := "User declined"

	pb := PodBinding{
		ID:              1,
		OrganizationID:  100,
		InitiatorPod:    "pod-init",
		TargetPod:       "pod-target",
		GrantedScopes:   pq.StringArray{BindingScopePodRead},
		PendingScopes:   pq.StringArray{BindingScopePodWrite},
		Status:          BindingStatusPending,
		RequestedAt:     &now,
		RejectionReason: &reason,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if pb.ID != 1 {
		t.Errorf("expected ID 1, got %d", pb.ID)
	}
	if pb.InitiatorPod != "pod-init" {
		t.Errorf("expected InitiatorPod 'pod-init', got %s", pb.InitiatorPod)
	}
	if pb.TargetPod != "pod-target" {
		t.Errorf("expected TargetPod 'pod-target', got %s", pb.TargetPod)
	}
}

// --- Benchmark Tests ---

func BenchmarkPodBindingHasScope(b *testing.B) {
	pb := &PodBinding{
		GrantedScopes: pq.StringArray{BindingScopePodRead, BindingScopePodWrite},
	}
	for i := 0; i < b.N; i++ {
		pb.HasScope(BindingScopePodRead)
	}
}

func BenchmarkPodBindingIsActive(b *testing.B) {
	pb := &PodBinding{Status: BindingStatusActive}
	for i := 0; i < b.N; i++ {
		pb.IsActive()
	}
}

func BenchmarkPodBindingCanObserve(b *testing.B) {
	pb := &PodBinding{
		Status:        BindingStatusActive,
		GrantedScopes: pq.StringArray{BindingScopePodRead},
	}
	for i := 0; i < b.N; i++ {
		pb.CanObserve()
	}
}
