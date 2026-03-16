package channel

import (
	"testing"

	"github.com/lib/pq"
)

// --- Test PodBinding Permission Methods ---

func TestPodBindingCanObserve(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		grantedScopes []string
		expected      bool
	}{
		{"active with read", BindingStatusActive, []string{BindingScopePodRead}, true},
		{"active without read", BindingStatusActive, []string{BindingScopePodWrite}, false},
		{"pending with read", BindingStatusPending, []string{BindingScopePodRead}, false},
		{"active with no scopes", BindingStatusActive, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := &PodBinding{
				Status:        tt.status,
				GrantedScopes: pq.StringArray(tt.grantedScopes),
			}
			if pb.CanObserve() != tt.expected {
				t.Errorf("expected CanObserve() = %v, got %v", tt.expected, pb.CanObserve())
			}
		})
	}
}

func TestPodBindingCanControl(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		grantedScopes []string
		expected      bool
	}{
		{"active with write", BindingStatusActive, []string{BindingScopePodWrite}, true},
		{"active without write", BindingStatusActive, []string{BindingScopePodRead}, false},
		{"pending with write", BindingStatusPending, []string{BindingScopePodWrite}, false},
		{"active with both", BindingStatusActive, []string{BindingScopePodRead, BindingScopePodWrite}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := &PodBinding{
				Status:        tt.status,
				GrantedScopes: pq.StringArray(tt.grantedScopes),
			}
			if pb.CanControl() != tt.expected {
				t.Errorf("expected CanControl() = %v, got %v", tt.expected, pb.CanControl())
			}
		})
	}
}
