package poddaemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildWindowsCmdLine(t *testing.T) {
	tests := []struct {
		name string
		path string
		args []string
		want string
	}{
		{
			name: "simple path no args",
			path: `C:\Windows\System32\cmd.exe`,
			args: nil,
			want: `C:\Windows\System32\cmd.exe`,
		},
		{
			name: "simple path with args",
			path: `cmd.exe`,
			args: []string{"/c", "echo", "hello"},
			want: `cmd.exe /c echo hello`,
		},
		{
			name: "path with spaces is quoted",
			path: `C:\Program Files\App\tool.exe`,
			args: []string{"--flag", "value"},
			want: `"C:\Program Files\App\tool.exe" --flag value`,
		},
		{
			name: "empty args slice",
			path: `tool.exe`,
			args: []string{},
			want: `tool.exe`,
		},
		{
			name: "arg with spaces is quoted",
			path: `python.exe`,
			args: []string{"-c", "print('hello world')"},
			want: `python.exe -c "print('hello world')"`,
		},
		{
			name: "arg with embedded quotes",
			path: `cmd.exe`,
			args: []string{"/c", `echo "hello"`},
			want: `cmd.exe /c "echo \"hello\""`,
		},
		{
			name: "empty arg is preserved",
			path: `cmd.exe`,
			args: []string{""},
			want: `cmd.exe ""`,
		},
		{
			name: "backslash before quote",
			path: `tool.exe`,
			args: []string{`path\with\"quote`},
			want: `tool.exe "path\with\\\"quote"`,
		},
		{
			name: "trailing backslash in path with spaces",
			path: `C:\My Dir\`,
			args: nil,
			want: `"C:\My Dir\\"`,
		},
		{
			name: "arg with tab",
			path: `tool.exe`,
			args: []string{"has\ttab"},
			want: "tool.exe \"has\ttab\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWindowsCmdLine(tt.path, tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQuoteWindowsArg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"", `""`},
		{"has space", `"has space"`},
		{`has"quote`, `"has\"quote"`},
		{`trail\`, `trail\`},
		{`trail with space\`, `"trail with space\\"`},
		{`a\\b`, `a\\b`},
		{`a\\"b`, `"a\\\\\"b"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := quoteWindowsArg(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
