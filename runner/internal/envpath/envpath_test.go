package envpath

import (
	"os"
	"os/exec"
	"runtime"
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
	if runtime.GOOS == "windows" {
		if !strings.Contains(strings.ToLower(result), "windows") {
			t.Errorf("expected PATH to contain windows system dir, got: %s", result)
		}
	} else {
		if !strings.Contains(result, "/usr/bin") {
			t.Errorf("expected PATH to contain /usr/bin, got: %s", result)
		}
	}
}

func TestResolveLoginShellPATH_FallbackOnEmptyShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping $SHELL test on Windows")
	}
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
	if runtime.GOOS == "windows" {
		t.Skip("skipping $SHELL test on Windows")
	}
	original := os.Getenv("SHELL")
	t.Setenv("SHELL", "/nonexistent/shell")
	defer os.Setenv("SHELL", original)

	expected := os.Getenv("PATH")
	result := ResolveLoginShellPATH()
	if result != expected {
		t.Errorf("expected fallback to current PATH %q, got %q", expected, result)
	}
}

// TestResolveLoginShellPATH_NoisyProfile verifies that a login shell whose
// profile prints extra output before/after the PATH does not corrupt the result.
// It creates a small sh wrapper that emits noise and then prints PATH with the
// sentinel, simulating a .zshrc with nvm or welcome-message output.
func TestMergeWithCurrentPATH_AppendsUniqueDirs(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/opt/nvm/bin:/usr/local/bin")

	result := mergeWithCurrentPATH("/usr/bin:/usr/local/bin")

	// /opt/nvm/bin should be appended (unique to current PATH)
	if !strings.Contains(result, "/opt/nvm/bin") {
		t.Errorf("expected merged PATH to include /opt/nvm/bin, got: %s", result)
	}
	// Resolved dirs should appear first
	if strings.Index(result, "/usr/bin") > strings.Index(result, "/opt/nvm/bin") {
		t.Error("expected resolved dirs to appear before appended dirs")
	}
}

func TestMergeWithCurrentPATH_NoDuplicates(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/usr/local/bin")

	result := mergeWithCurrentPATH("/usr/bin:/usr/local/bin")

	dirs := strings.Split(result, ":")
	seen := make(map[string]bool)
	for _, d := range dirs {
		if seen[d] {
			t.Errorf("duplicate dir in merged PATH: %s", d)
		}
		seen[d] = true
	}
}

func TestMergeWithCurrentPATH_EmptyCurrentPATH(t *testing.T) {
	t.Setenv("PATH", "")

	resolved := "/usr/bin:/usr/local/bin"
	result := mergeWithCurrentPATH(resolved)
	if result != resolved {
		t.Errorf("expected %q, got %q", resolved, result)
	}
}

func TestMergeWithCurrentPATH_IdenticalPATH(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/usr/local/bin")

	resolved := "/usr/bin:/usr/local/bin"
	result := mergeWithCurrentPATH(resolved)
	if result != resolved {
		t.Errorf("expected %q when paths are identical, got %q", resolved, result)
	}
}

func TestResolveLoginShellPATH_NoisyProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping login shell test on Windows")
	}
	// Create a fake login shell script that emits noisy output around the PATH.
	dir := t.TempDir()
	fakeShell := dir + "/fakesh"
	script := `#!/bin/sh
# Simulate a noisy profile: emit text before and after PATH
echo "Welcome to FakeShell!"
echo "nvm initialized"
# Execute the command passed via -c, which will print the sentinel line
eval "$3"
echo "Done."
`
	if err := os.WriteFile(fakeShell, []byte(script), 0700); err != nil {
		t.Fatalf("failed to write fake shell: %v", err)
	}

	// Verify the fake shell is executable.
	if _, err := exec.LookPath(fakeShell); err != nil {
		t.Skip("fake shell not executable, skipping")
	}

	t.Setenv("SHELL", fakeShell)
	t.Setenv("PATH", "/usr/bin:/bin:/usr/local/bin")

	result := ResolveLoginShellPATH()
	if result == "" {
		t.Fatal("expected non-empty PATH")
	}
	if !strings.Contains(result, ":") {
		t.Errorf("resolved PATH looks invalid (no colon): %q", result)
	}
	// Must not contain the noise text.
	if strings.Contains(result, "Welcome") || strings.Contains(result, "nvm") || strings.Contains(result, "Done") {
		t.Errorf("noisy profile output leaked into resolved PATH: %q", result)
	}
}
