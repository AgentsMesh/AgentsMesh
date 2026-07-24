package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

func TestDispatchOutputHandlerPaths(t *testing.T) {
	tests := []struct {
		name  string
		delay time.Duration
	}{
		{name: "fast"},
		{name: "slow warning", delay: handlerSlowWarnThreshold + 10*time.Millisecond},
		{name: "slow error", delay: handlerSlowErrorThreshold + 10*time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received []byte
			terminal := &Terminal{label: "dispatch-test"}
			terminal.onOutput = func(data []byte) {
				time.Sleep(tt.delay)
				received = append(received, data...)
			}

			terminal.dispatchOutput([]byte("payload"), 7)
			if got := string(received); got != "payload" {
				t.Fatalf("handler received %q, want payload", got)
			}
		})
	}
}

func TestDispatchOutputWithoutHandlerReturns(t *testing.T) {
	terminal := &Terminal{label: "no-handler"}
	terminal.dispatchOutput([]byte("discarded"), 1)
}

func TestWatchOutputHandlerCompletionAndBlockedDump(t *testing.T) {
	terminal := &Terminal{label: fmt.Sprintf("watchdog-test-%d-%d", os.Getpid(), time.Now().UnixNano())}

	done := make(chan struct{})
	close(done)
	terminal.watchOutputHandler(done, time.Now(), 1, 2)

	pattern := filepath.Join(config.TempBaseDir(), "blocked-"+terminal.label+"-*.stacks")
	before, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]struct{}, len(before))
	for _, path := range before {
		seen[path] = struct{}{}
	}

	terminal.watchOutputHandler(make(chan struct{}), time.Now(), 3, 4)

	after, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	var created string
	for _, path := range after {
		if _, existed := seen[path]; !existed {
			created = path
			break
		}
	}
	if created == "" {
		t.Fatalf("blocked handler did not create stack dump matching %q", pattern)
	}
	t.Cleanup(func() { _ = os.Remove(created) })
	contents, err := os.ReadFile(created)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "TestWatchOutputHandlerCompletionAndBlockedDump") {
		t.Fatalf("stack dump does not identify blocked caller: %s", contents)
	}
}
