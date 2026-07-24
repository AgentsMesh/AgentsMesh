package mockagent

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
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

func TestRunPTYTerminalRenderEntrypointHandlesImmediateEOF(t *testing.T) {
	input, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outputReader, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldInput, oldOutput := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = input, output
	t.Cleanup(func() {
		os.Stdin, os.Stdout = oldInput, oldOutput
		_ = input.Close()
		_ = output.Close()
		_ = outputReader.Close()
	})
	_ = inputWriter.Close()

	if code := RunPTY("terminal_render", slog.Default()); code != 0 {
		t.Fatalf("RunPTY exit code = %d", code)
	}
	_ = output.Close()
	data, err := io.ReadAll(outputReader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), terminalRenderReady) {
		t.Fatalf("terminal render entrypoint output = %q", data)
	}
}

func TestRunPTYWithIORenderScenarios(t *testing.T) {
	for _, tt := range []struct {
		name     string
		scenario string
		input    string
		marker   string
	}{
		{name: "render", scenario: "terminal_render", input: terminalRenderCommand + "\n", marker: "E2E_RENDER_DONE"},
		{name: "alternate snapshot", scenario: "terminal_alt_snapshot", input: terminalAltSnapshotEnterCommand + "\n", marker: terminalAltSnapshotActive},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			if code := runPTYWithIO(tt.scenario, strings.NewReader(tt.input), &out, nil, slog.Default()); code != 0 {
				t.Fatalf("exit code = %d", code)
			}
			if !strings.Contains(out.String(), tt.marker) {
				t.Fatalf("output missing %q", tt.marker)
			}
		})
	}
}

func TestTerminalRenderLoop_WaitsForCommandAndEmitsFragmentedFixture(t *testing.T) {
	in := strings.NewReader("ignored\nrender-probe\nrender-probe\n")
	var out bytes.Buffer

	terminalRenderLoop(in, &out, 0, 0)

	got := out.String()
	wantSubs := []string{
		terminalRenderReady + "\r\n",
		terminalNormalBufferSentinel + "\r\n",
		"\x1b[?1049h",
		"ALT-BUFFER-PROBE\r\n",
		"\x1b[?1049l",
		"\x1b]0;AgentsMesh terminal E2E\x07",
		"\x1bPignored-dcs-payload\x1b\\",
		"SCROLL-00|abcdefghijklmnopqrstuvwxyz|\r\n",
		"SCROLL-47|abcdefghijklmnopqrstuvwxyz|\r\n",
		"\x1b[32mOFFSET:\x1b[0m\x1b[35m☰\x1b[0m|\x1b[36mSELECT-ME\x1b[0m|TAIL\r\n",
		"\x1b[31mCOMB:A\x1b[0m\x1b[32me\u0301\x1b[0m" +
			"\x1b[33mB|CJK:A\x1b[0m\x1b[34m界\x1b[0m" +
			"\x1b[35mB|EMOJI:A\x1b[0m\x1b[36m🫠\x1b[0m" +
			"\x1b[31mB|ZWJ:A\x1b[0m\x1b[32m👩‍💻\x1b[0m" +
			"\x1b[33mB|VS:A\x1b[0m\x1b[34m♥️\x1b[0m" +
			"\x1b[35mB|END\x1b[0m\r\n",
		"CURSOR:xxxxx\x1b[5DOK\x1b[3C\r\n",
		"E2E_RENDER_DONE\r\n",
	}
	for _, want := range wantSubs {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- full output ---\n%q", want, got)
		}
	}
	if strings.Count(got, "E2E_RENDER_DONE") != 1 {
		t.Fatalf("fixture rendered more than once: %q", got)
	}
}

func TestTerminalRenderLoopEarlyEOFAndClosedResizeChannel(t *testing.T) {
	var early bytes.Buffer
	terminalRenderLoop(strings.NewReader(""), &early, 0, 0)
	if early.String() != terminalRenderReady+"\r\n" {
		t.Fatalf("early EOF output = %q", early.String())
	}

	reader, writer := io.Pipe()
	resizeSignals := make(chan os.Signal)
	var out bytes.Buffer
	baseline := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		terminalRenderLoopWithResize(reader, &out, 0, 0, resizeSignals, func() (int, int, error) {
			baseline <- struct{}{}
			return 80, 24, nil
		})
		close(done)
	}()
	if _, err := io.WriteString(writer, terminalRenderCommand+"\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-baseline:
	case <-time.After(time.Second):
		t.Fatal("render loop did not emit baseline size")
	}
	close(resizeSignals)
	time.Sleep(10 * time.Millisecond)
	_ = writer.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("render loop did not exit")
	}

	if _, _, err := stdoutTerminalSize(); err == nil {
		t.Log("stdout happened to be a terminal")
	}
	writeTerminalRenderFixture(io.Discard, 0, time.Nanosecond)
	writeTerminalChunks(io.Discard, time.Nanosecond, []byte("delayed"))
}

func TestWatchTerminalResizeLifecycle(t *testing.T) {
	signals, stop := watchTerminalResize()
	if signals == nil {
		t.Fatal("Unix resize watcher returned nil channel")
	}
	stop()
}

func TestTerminalRenderLoop_EmitsChangedPTYSizeAfterResizeSignal(t *testing.T) {
	reader, writer := io.Pipe()
	resizeSignals := make(chan os.Signal, 2)
	sizes := make(chan [2]int, 3)
	sizeReads := make(chan struct{}, 3)
	readSize := func() (int, int, error) {
		size := <-sizes
		sizeReads <- struct{}{}
		return size[0], size[1], nil
	}
	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		terminalRenderLoopWithResize(reader, &out, 0, 0, resizeSignals, readSize)
	}()

	waitForSizeRead := func() {
		t.Helper()
		select {
		case <-sizeReads:
		case <-time.After(2 * time.Second):
			t.Fatal("terminal render loop did not read the PTY size")
		}
	}

	sizes <- [2]int{120, 32}
	if _, err := io.WriteString(writer, terminalRenderCommand+"\n"); err != nil {
		t.Fatalf("write render command: %v", err)
	}
	waitForSizeRead()

	// Duplicate SIGWINCH notifications for the same size must not create noise.
	sizes <- [2]int{120, 32}
	resizeSignals <- os.Interrupt
	waitForSizeRead()

	sizes <- [2]int{96, 28}
	resizeSignals <- os.Interrupt
	waitForSizeRead()
	if err := writer.Close(); err != nil {
		t.Fatalf("close render input: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("terminal render loop did not stop after input closed")
	}

	got := out.String()
	baseline := terminalRenderSizePrefix + ":001:120x32\r\n"
	changed := terminalRenderSizePrefix + ":002:96x28\r\n"
	if strings.Count(got, baseline) != 1 {
		t.Fatalf("baseline size marker count = %d, want 1; output=%q", strings.Count(got, baseline), got)
	}
	if !strings.Contains(got, changed) {
		t.Fatalf("changed size marker missing %q; output=%q", changed, got)
	}
}

func TestTerminalAltSnapshotLoop_HoldsActiveBufferUntilExitCommand(t *testing.T) {
	in := strings.NewReader("ignored\nenter-alt-buffer\n")
	var out bytes.Buffer

	terminalAltSnapshotLoop(in, &out, 0)

	got := out.String()
	wantSubs := []string{
		terminalAltSnapshotReady + "\r\n",
		terminalNormalBufferSentinel + "\r\n",
		"ALT-NORMAL-SCROLL-00|abcdefghijklmnopqrstuvwxyz|\r\n",
		"ALT-NORMAL-SCROLL-47|abcdefghijklmnopqrstuvwxyz|\r\n",
		"\x1b[?1049h",
		terminalAltSnapshotActive + "\r\n",
		terminalAltSnapshotSurface + "\r\n",
	}
	for _, want := range wantSubs {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n--- full output ---\n%q", want, got)
		}
	}
	if strings.Contains(got, "\x1b[?1049l") || strings.Contains(got, terminalAltSnapshotExited) {
		t.Fatalf("alternate buffer exited without an explicit command: %q", got)
	}
}

func TestTerminalAltSnapshotLoop_ExitsOnceAndRestoresNormalBuffer(t *testing.T) {
	in := strings.NewReader("exit-alt-buffer\nenter-alt-buffer\nexit-alt-buffer\nenter-alt-buffer\nexit-alt-buffer\n")
	var out bytes.Buffer

	terminalAltSnapshotLoop(in, &out, 0)

	got := out.String()
	enter := strings.Index(got, "\x1b[?1049h")
	active := strings.Index(got, terminalAltSnapshotActive)
	exit := strings.Index(got, "\x1b[?1049l")
	exited := strings.Index(got, terminalAltSnapshotExited)
	if enter < 0 || active < enter || exit < active || exited < exit {
		t.Fatalf("alternate-buffer protocol is out of order: %q", got)
	}
	if strings.Count(got, "\x1b[?1049h") != 1 || strings.Count(got, "\x1b[?1049l") != 1 {
		t.Fatalf("alternate-buffer transition must be one-shot: %q", got)
	}
	if strings.Count(got, terminalAltSnapshotExited) != 1 {
		t.Fatalf("exit marker count = %d, want 1: %q", strings.Count(got, terminalAltSnapshotExited), got)
	}
}

// The autopilot scenario must echo `got: <line>` then leave a prompt symbol as
// the trailing screen line so the runner's PTY state detector classifies the
// pod as waiting. Without it the AutopilotController never fires an iteration.
// Tested via promptEchoLoop directly (turnDelay=0) to avoid the production
// turn delay that holds the pod executing past the controller's MinTriggerGap.
func TestPromptEchoLoop_EchoesThenPrompt(t *testing.T) {
	in := strings.NewReader("echo step1\necho step2\n")
	var out bytes.Buffer
	promptEchoLoop(in, &out, 0)
	got := out.String()
	if !strings.Contains(got, "got: echo step1\n") || !strings.Contains(got, "got: echo step2\n") {
		t.Errorf("missing echo round-trips: %q", got)
	}
	if !strings.HasSuffix(got, autopilotPrompt) {
		t.Errorf("output must end with prompt %q, got tail %q", autopilotPrompt, got)
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
