package vt

import (
	"bytes"
	"testing"
	"time"
)

func TestSnapshotParserPrefixIsBounded(t *testing.T) {
	tests := []struct {
		name       string
		introducer string
		bodyByte   byte
		discard    string
		terminator string
	}{
		{name: "CSI", introducer: "\x1b[", bodyByte: '1', discard: "\x1b[!!", terminator: "m"},
		{name: "OSC", introducer: "\x1b]", bodyByte: 'x', discard: "\x1b]999999;", terminator: "\a"},
		{name: "DCS", introducer: "\x1bP", bodyByte: 'x', discard: "\x1bP999999;z", terminator: "\x1b\\"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			terminal := NewVirtualTerminal(20, 4, 100)
			input := append([]byte(test.introducer), bytes.Repeat(
				[]byte{test.bodyByte}, maxParserSequenceBytes,
			)...)
			terminal.Feed(input)

			prefix := terminal.GetSnapshot().ParserPrefix
			if len(prefix) != maxParserSequenceBytes+2 {
				t.Fatalf("bounded prefix length = %d, want %d", len(prefix), maxParserSequenceBytes+2)
			}

			terminal.Feed([]byte{test.bodyByte})
			discard := terminal.GetSnapshot().ParserPrefix
			if got := string(intBytes(discard)); got != test.discard {
				t.Fatalf("discard prefix = %q, want %q", got, test.discard)
			}

			restored := NewVirtualTerminal(20, 4, 100)
			restored.Feed(intBytes(discard))
			continuation := test.terminator + "OK"
			terminal.Feed([]byte(continuation))
			restored.Feed([]byte(continuation))
			if got := terminal.GetDisplay(); got != "OK" || got != restored.GetDisplay() {
				t.Fatalf("parser recovery = %q, restored %q", got, restored.GetDisplay())
			}
		})
	}
}

func TestOversizedOSCNeverInvokesHandler(t *testing.T) {
	terminal := NewVirtualTerminal(20, 4, 100)
	called := make(chan struct{}, 1)
	terminal.SetOSCHandler(func(_ int, _ []string) { called <- struct{}{} })

	payload := append([]byte("\x1b]9;"), bytes.Repeat(
		[]byte{'x'}, maxParserSequenceBytes,
	)...)
	payload = append(payload, '\a')
	terminal.Feed(payload)
	if got := terminal.GetDisplay(); got != "" {
		t.Fatalf("oversized OSC body leaked onto screen: %q", got)
	}
	select {
	case <-called:
		t.Fatal("oversized OSC invoked its handler")
	case <-time.After(50 * time.Millisecond):
	}
}
