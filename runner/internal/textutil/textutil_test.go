package textutil

import "testing"

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"unix only", "a\nb\nc", "a\nb\nc"},
		{"windows", "a\r\nb\r\nc", "a\nb\nc"},
		{"mixed", "a\r\nb\nc\r\n", "a\nb\nc\n"},
		{"bare cr", "a\rb\rc", "a\rb\rc"}, // \r alone is preserved
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLineEndings(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeLineEndings(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // expected line count
	}{
		{"empty", "", 1},
		{"single line", "hello", 1},
		{"unix lines", "a\nb\nc", 3},
		{"windows lines", "a\r\nb\r\nc", 3},
		{"trailing newline", "a\nb\n", 3}, // last element is ""
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := SplitLines(tt.input)
			if len(lines) != tt.want {
				t.Errorf("SplitLines(%q) got %d lines, want %d", tt.input, len(lines), tt.want)
			}
		})
	}
}
