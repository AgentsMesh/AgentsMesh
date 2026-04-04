//go:build integration

package runner

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func buildTestPod(t *testing.T, agentfile string, opts ...func(*runnerv1.CreatePodCommand)) *Pod {
	t.Helper()
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "lifecycle-" + t.Name(),
		AgentfileSource: agentfile,
	}
	for _, o := range opts {
		o(cmd)
	}
	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	return pod
}

func TestPodLifecycle_EchoCommand_Integration(t *testing.T) {
	af := "AGENT echo\nMODE pty\nPROMPT_POSITION prepend\n"
	pod := buildTestPod(t, af, func(c *runnerv1.CreatePodCommand) {
		c.InitialPrompt = "hello from integration test"
	})
	comps := testPTYComponents(pod)
	defer comps.Terminal.Stop()

	var mu sync.Mutex
	var output []byte
	doneCh := make(chan struct{})

	comps.Terminal.SetOutputHandler(func(data []byte) {
		mu.Lock()
		output = append(output, data...)
		mu.Unlock()
	})
	comps.Terminal.SetExitHandler(func(_ int) { close(doneCh) })

	pid := pod.IO.GetPID()
	if err := comps.Terminal.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if startPID := pod.IO.GetPID(); startPID <= 0 {
		t.Errorf("PID after Start = %d, want > 0", startPID)
	}
	// pid before Start may be 0, that's expected
	_ = pid

	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for echo to exit")
	}

	mu.Lock()
	got := string(output)
	mu.Unlock()
	if !strings.Contains(got, "hello from integration test") {
		t.Errorf("output = %q, want to contain %q", got, "hello from integration test")
	}
}

func TestPodLifecycle_CatInteractive_Integration(t *testing.T) {
	pod := buildTestPod(t, "AGENT cat\nMODE pty\nPROMPT_POSITION prepend\n")
	comps := testPTYComponents(pod)
	defer comps.Terminal.Stop()

	comps.Terminal.SetOutputHandler(func(data []byte) {
		comps.VirtualTerminal.Feed(data)
	})
	if err := comps.Terminal.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Send input — cat echoes back via PTY
	if err := pod.IO.SendInput("test input\n"); err != nil {
		t.Fatalf("SendInput failed: %v", err)
	}

	// Wait for VT to accumulate echoed output
	deadline := time.After(3 * time.Second)
	for {
		snapshot := comps.VirtualTerminal.GetScreenSnapshot()
		if strings.Contains(snapshot, "test input") {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("snapshot never contained 'test input', got: %q",
				comps.VirtualTerminal.GetScreenSnapshot())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func TestPodLifecycle_AgentFileEval_BuildIntegration(t *testing.T) {
	agentfile := `AGENT echo
MODE pty
PROMPT_POSITION prepend
ENV TEST_INTEGRATION_VAR TEXT OPTIONAL
`
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "agentfile-eval-test",
		AgentfileSource: agentfile,
		ConfigValues:  map[string]string{"test_key": "test_value"},
	}
	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer func() {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}()

	if pod.LaunchCommand != "echo" {
		t.Errorf("LaunchCommand = %q, want %q", pod.LaunchCommand, "echo")
	}
	if pod.InteractionMode != InteractionModePTY {
		t.Errorf("InteractionMode = %q, want %q", pod.InteractionMode, InteractionModePTY)
	}
	comps := testPTYComponents(pod)
	if comps == nil || comps.Terminal == nil {
		t.Error("Terminal should not be nil for PTY pod")
	}
	if comps == nil || comps.VirtualTerminal == nil {
		t.Error("VirtualTerminal should not be nil for PTY pod")
	}
}

func TestPodLifecycle_ACP_Build_Integration(t *testing.T) {
	agentfile := `AGENT echo
MODE acp
MODE acp "flag"
PROMPT_POSITION prepend
`
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "acp-build-test",
		AgentfileSource: agentfile,
	}
	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if pod.InteractionMode != InteractionModeACP {
		t.Errorf("InteractionMode = %q, want %q", pod.InteractionMode, InteractionModeACP)
	}
	if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
		t.Error("Terminal should be nil for ACP pod")
	}
	if pod.LaunchCommand != "echo" {
		t.Errorf("LaunchCommand = %q, want %q", pod.LaunchCommand, "echo")
	}
	// MODE acp "flag" should put "flag" into LaunchArgs
	found := false
	for _, a := range pod.LaunchArgs {
		if a == "flag" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("LaunchArgs = %v, want to contain %q", pod.LaunchArgs, "flag")
	}
}
