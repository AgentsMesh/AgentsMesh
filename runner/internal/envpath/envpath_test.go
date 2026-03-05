package envpath

import (
	"os"
	"strings"
	"testing"
)

func TestResolveLoginShellPATH_ReturnsNonEmpty(t *testing.T) {
	result := ResolveLoginShellPATH()
	if result == "" {
		t.Fatal("expected non-empty PATH")
	}
}

func TestResolveLoginShellPATH_ContainsStandardDirs(t *testing.T) {
	result := ResolveLoginShellPATH()
	if !strings.Contains(result, "/usr/bin") {
		t.Errorf("expected PATH to contain /usr/bin, got: %s", result)
	}
}

func TestResolveLoginShellPATH_FallbackOnEmptyShell(t *testing.T) {
	original := os.Getenv("SHELL")
	t.Setenv("SHELL", "")
	defer os.Setenv("SHELL", original)

	expected := os.Getenv("PATH")
	result := ResolveLoginShellPATH()
	if result != expected {
		t.Errorf("expected fallback to current PATH %q, got %q", expected, result)
	}
}

func TestResolveLoginShellPATH_FallbackOnInvalidShell(t *testing.T) {
	original := os.Getenv("SHELL")
	t.Setenv("SHELL", "/nonexistent/shell")
	defer os.Setenv("SHELL", original)

	expected := os.Getenv("PATH")
	result := ResolveLoginShellPATH()
	if result != expected {
		t.Errorf("expected fallback to current PATH %q, got %q", expected, result)
	}
}
