package mockagent

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestRunPTY_EchoScenario(t *testing.T) {
	in := strings.NewReader("hello\nworld\n")
	var out bytes.Buffer

	code := runPTYWithIO("echo", in, &out, nil, slog.Default())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	got := out.String()
	wantSubs := []string{"ready\n", "got: hello\n", "got: world\n"}
	for _, w := range wantSubs {
		if !strings.Contains(got, w) {
			t.Errorf("output missing %q\n--- full output ---\n%s", w, got)
		}
	}
}

func TestRunPTY_EmptyStdin(t *testing.T) {
	var out bytes.Buffer
	code := runPTYWithIO("echo", strings.NewReader(""), &out, nil, slog.Default())
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "ready") {
		t.Errorf("output missing ready signal: %q", out.String())
	}
}

func TestRunPTY_UnknownScenario(t *testing.T) {
	var out bytes.Buffer
	code := runPTYWithIO("nonexistent", strings.NewReader(""), &out, nil, slog.Default())
	if code == 0 {
		t.Error("expected non-zero exit for unknown scenario")
	}
}

func TestEchoLoop_PreservesLineOrder(t *testing.T) {
	in := strings.NewReader("a\nb\nc\n")
	var out bytes.Buffer
	echoLoop(in, &out)
	want := "got: a\ngot: b\ngot: c\n"
	if out.String() != want {
		t.Errorf("got %q, want %q", out.String(), want)
	}
}

// Ensures the echo loop tolerates very large lines (matches the 1MB scanner
// buffer in runner/internal/acp/reader.go used by real agents).
func TestEchoLoop_LargeLine(t *testing.T) {
	big := strings.Repeat("x", 100_000)
	in := strings.NewReader(big + "\n")
	var out bytes.Buffer
	echoLoop(in, &out)
	if !strings.Contains(out.String(), "got: "+big[:50]) {
		t.Error("large line was not echoed")
	}
}

var _ io.Reader = (*strings.Reader)(nil)
