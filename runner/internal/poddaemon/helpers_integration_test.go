//go:build integration

package poddaemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
)

// TestMain wires the test process so it can both initialize processmgr (every
// integration test that drives CreateSession lands in startDaemon, which
// requires processmgr.Global()) and act as the launcher subprocess when
// startDaemon re-execs the test binary itself via os.Executable(). Without
// the LauncherSubcommand fork the first test that calls CreateSession would
// fail to spawn a daemon.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == processmgr.LauncherSubcommand {
		processmgr.RunLauncher()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	processmgr.Init(ctx, processmgr.Options{})
	os.Exit(m.Run())
}

// findModuleRoot walks up from the current directory to find go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}

// buildTestRunner returns a usable path to the runner binary. Two execution
// contexts are supported:
//
//  1. Bazel test sandbox: TEST_SRCDIR is set and the runner binary is wired
//     in as a data dependency. Resolve via runfiles layout.
//
//  2. Plain `go test -tags=integration` from a developer shell: fall back to
//     a previously-built bazel-bin artifact, then to `bazel build` as a
//     last resort. We deliberately do not invoke `go build` because the
//     monorepo's proto stubs live in bazel-bin/ and are unreachable from
//     `go build` without manual generation.
func buildTestRunner(t *testing.T) string {
	t.Helper()

	if path := runnerFromRunfiles(); path != "" {
		return path
	}

	modRoot := findModuleRoot(t)
	if path := runnerFromBazelBin(modRoot); path != "" {
		return path
	}

	cmd := exec.Command("bazel", "build", "//runner/cmd/runner:runner")
	cmd.Dir = modRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Skipf("cannot locate runner binary: bazel build failed (%v) and no prebuilt artifact found", err)
	}
	if path := runnerFromBazelBin(modRoot); path != "" {
		return path
	}
	t.Skip("runner binary still not found after bazel build — environment is incompatible")
	return ""
}

func runnerFromRunfiles() string {
	srcdir := os.Getenv("TEST_SRCDIR")
	if srcdir == "" {
		return ""
	}
	// Bzlmod main repo name is "_main"; the binary is at
	// _main/runner/cmd/runner/runner_/<name>.
	bin := "runner"
	if runtime.GOOS == "windows" {
		bin = "runner.exe"
	}
	candidate := filepath.Join(srcdir, "_main", "runner", "cmd", "runner", "runner_", bin)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func runnerFromBazelBin(modRoot string) string {
	// `bazel cquery --output=files` would be the canonical lookup but it is
	// expensive to invoke for every test. The conventional layout is stable
	// enough to glob.
	bin := "runner"
	if runtime.GOOS == "windows" {
		bin = "runner.exe"
	}
	matches, _ := filepath.Glob(filepath.Join(modRoot, "bazel-out", "*", "bin", "runner", "cmd", "runner", "runner_", bin))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// shortWorkspace creates a short temp dir for integration tests.
// Returns (workspace, sandbox) paths.
func shortWorkspace(t *testing.T, name string) (string, string) {
	t.Helper()

	workspace := t.TempDir()
	sandbox := filepath.Join(workspace, name)
	require.NoError(t, os.MkdirAll(sandbox, 0755))
	return workspace, sandbox
}
