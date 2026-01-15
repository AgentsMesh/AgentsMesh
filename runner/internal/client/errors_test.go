package client

import (
	"testing"
)

func TestPodErrorConstants(t *testing.T) {
	// Verify all error codes are defined correctly
	expectedCodes := map[string]string{
		"ErrCodeUnknown":         ErrCodeUnknown,
		"ErrCodeSandboxCreate":   ErrCodeSandboxCreate,
		"ErrCodeGitClone":        ErrCodeGitClone,
		"ErrCodeGitWorktree":     ErrCodeGitWorktree,
		"ErrCodeGitAuth":         ErrCodeGitAuth,
		"ErrCodeFileCreate":      ErrCodeFileCreate,
		"ErrCodeFilePermission":  ErrCodeFilePermission,
		"ErrCodeCommandNotFound": ErrCodeCommandNotFound,
		"ErrCodeCommandStart":    ErrCodeCommandStart,
		"ErrCodeWorkDirNotExist": ErrCodeWorkDirNotExist,
		"ErrCodeDiskFull":        ErrCodeDiskFull,
	}

	// Just verify they're non-empty strings
	for name, code := range expectedCodes {
		if code == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestPodErrorError(t *testing.T) {
	err := &PodError{
		Code:    ErrCodeCommandNotFound,
		Message: "command 'claude' not found",
	}

	if err.Error() != "command 'claude' not found" {
		t.Errorf("Error(): got %q, want %q", err.Error(), "command 'claude' not found")
	}
}

func TestNewPodError(t *testing.T) {
	err := NewPodError(ErrCodeGitClone, "git clone failed")

	if err.Code != ErrCodeGitClone {
		t.Errorf("Code: got %q, want %q", err.Code, ErrCodeGitClone)
	}
	if err.Message != "git clone failed" {
		t.Errorf("Message: got %q, want %q", err.Message, "git clone failed")
	}
	if err.Details != nil {
		t.Errorf("Details should be nil, got %v", err.Details)
	}
}

func TestNewPodErrorWithDetails(t *testing.T) {
	details := map[string]string{
		"path":   "/tmp/repo",
		"reason": "permission denied",
	}
	err := NewPodErrorWithDetails(ErrCodeFileCreate, "failed to create file", details)

	if err.Code != ErrCodeFileCreate {
		t.Errorf("Code: got %q, want %q", err.Code, ErrCodeFileCreate)
	}
	if err.Message != "failed to create file" {
		t.Errorf("Message: got %q, want %q", err.Message, "failed to create file")
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
	if err.Details["path"] != "/tmp/repo" {
		t.Errorf("Details[path]: got %q, want %q", err.Details["path"], "/tmp/repo")
	}
	if err.Details["reason"] != "permission denied" {
		t.Errorf("Details[reason]: got %q, want %q", err.Details["reason"], "permission denied")
	}
}

func TestCreatePodResponse(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		resp := CreatePodResponse{
			Success: true,
			PodKey:  "pod-123",
		}

		if !resp.Success {
			t.Error("Success should be true")
		}
		if resp.PodKey != "pod-123" {
			t.Errorf("PodKey: got %q, want %q", resp.PodKey, "pod-123")
		}
		if resp.Error != nil {
			t.Error("Error should be nil for success response")
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := CreatePodResponse{
			Success: false,
			Error: &PodError{
				Code:    ErrCodeCommandNotFound,
				Message: "command not found",
			},
		}

		if resp.Success {
			t.Error("Success should be false")
		}
		if resp.PodKey != "" {
			t.Errorf("PodKey should be empty, got %q", resp.PodKey)
		}
		if resp.Error == nil {
			t.Error("Error should not be nil for error response")
		}
		if resp.Error.Code != ErrCodeCommandNotFound {
			t.Errorf("Error.Code: got %q, want %q", resp.Error.Code, ErrCodeCommandNotFound)
		}
	})
}
