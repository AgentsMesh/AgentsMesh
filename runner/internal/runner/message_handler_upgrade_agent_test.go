package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
)

func getUpgradeAgentResults(mockConn *client.MockConnection) []*runnerv1.UpgradeAgentResultEvent {
	var results []*runnerv1.UpgradeAgentResultEvent
	for _, e := range mockConn.GetEvents() {
		if e.Type == "upgrade_agent_result" {
			if evt, ok := e.Data.(*runnerv1.UpgradeAgentResultEvent); ok {
				results = append(results, evt)
			}
		}
	}
	return results
}

func TestOnUpgradeAgent_EmptyArgv_Failed(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	if err := handler.OnUpgradeAgent(&runnerv1.UpgradeAgentCommand{
		RequestId: "req-1",
		AgentSlug: "claude-code",
	}); err != nil {
		t.Fatalf("OnUpgradeAgent should not return error: %v", err)
	}
	results := getUpgradeAgentResults(mockConn)
	if len(results) != 1 || results[0].Status != "failed" {
		t.Fatalf("expected one failed result, got %+v", results)
	}
}

func TestOnUpgradeAgent_ToolNotFound_Failed(t *testing.T) {
	r := newTestRunnerForUpgrade(0)
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	if err := handler.OnUpgradeAgent(&runnerv1.UpgradeAgentCommand{
		RequestId:   "req-2",
		AgentSlug:   "claude-code",
		UpgradeArgv: []string{"nonexistent-upgrade-tool-xyz-123"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	results := getUpgradeAgentResults(mockConn)
	if len(results) != 1 || results[0].Status != "failed" {
		t.Fatalf("expected one failed result, got %+v", results)
	}
}

func TestOnUpgradeAgent_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake upgrade tool is a unix shell script")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "fake-upgrade")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake tool: %v", err)
	}

	r := newTestRunnerForUpgrade(0)
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	if err := handler.OnUpgradeAgent(&runnerv1.UpgradeAgentCommand{
		RequestId:   "req-3",
		AgentSlug:   "claude-code",
		UpgradeArgv: []string{tool},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	results := getUpgradeAgentResults(mockConn)
	if len(results) != 1 || results[0].Status != "success" {
		t.Fatalf("expected one success result, got %+v", results)
	}
	if results[0].AgentSlug != "claude-code" {
		t.Fatalf("expected agent_slug claude-code, got %q", results[0].AgentSlug)
	}
}

func TestOnUpgradeAgent_ReprobesVersionAfterUpgrade(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake tools are unix shell scripts")
	}
	dir := t.TempDir()
	tool := filepath.Join(dir, "upgrade-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Fake agent executable whose --version reports the post-upgrade version.
	exe := filepath.Join(dir, "fakeagent")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho 'fakeagent v2.5.0'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := newTestRunnerForUpgrade(0)
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	if err := handler.OnUpgradeAgent(&runnerv1.UpgradeAgentCommand{
		RequestId:   "req-reprobe",
		AgentSlug:   "fakeagent",
		Executable:  exe,
		UpgradeArgv: []string{tool},
	}); err != nil {
		t.Fatal(err)
	}
	results := getUpgradeAgentResults(mockConn)
	if len(results) != 1 || results[0].Status != "success" {
		t.Fatalf("expected success, got %+v", results)
	}
	if results[0].NewVersion != "2.5.0" {
		t.Fatalf("expected re-probed version 2.5.0, got %q", results[0].NewVersion)
	}
}

func TestOnUpgradeAgent_SerializesConcurrentUpgrades(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake tool is a unix shell script")
	}
	dir := t.TempDir()
	logFile := filepath.Join(dir, "exec.log")
	// Each run appends S, sleeps, appends E. Serialized → "SESE"; if the mutex
	// failed to serialize, the two runs interleave → "SSEE".
	tool := filepath.Join(dir, "slow-tool")
	script := fmt.Sprintf("#!/bin/sh\nprintf S >> %q\nsleep 0.1\nprintf E >> %q\n", logFile, logFile)
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	r := newTestRunnerForUpgrade(0)
	mockConn := client.NewMockConnection()
	handler := NewRunnerMessageHandler(r, r.podStore, mockConn)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = handler.OnUpgradeAgent(&runnerv1.UpgradeAgentCommand{
				RequestId:   "concurrent",
				AgentSlug:   "x",
				UpgradeArgv: []string{tool},
			})
		}()
	}
	wg.Wait()

	log, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(log) != "SESE" {
		t.Fatalf("expected serialized execution \"SESE\", got %q (agentUpgradeMu not serializing)", log)
	}
}
