package displaykit

import (
	"errors"
	"testing"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"plain", "Hello", "Hello"},
		{"trim_outer", "  hello  ", "hello"},
		{"collapse_spaces", "a    b", "a b"},
		{"newline_collapsed", "line1\nline2", "line1 line2"},
		{"tab_collapsed", "a\t\t\tb", "a b"},
		{"zero_width_stripped", "ali\u200Bce", "alice"},
		{"rtl_override_stripped", "user\u202Eadmin", "useradmin"},
		{"bom_stripped", "\uFEFFhello", "hello"},
		{"control_stripped", "ab\x01\x02cd", "abcd"},
		{"empty_after_sanitize", "\u200B\u200B   ", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Sanitize(c.in); got != c.want {
				t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestSanitizeAndValidate(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		minLen  int
		maxLen  int
		want    string
		wantErr error
	}{
		{"valid", "Hello", 1, 100, "Hello", nil},
		{"empty_after_strip", "\u200B", 1, 100, "", ErrEmpty},
		{"too_long", "abcdef", 1, 5, "", ErrTooLong},
		{"sanitize_then_validate", "  hi  ", 1, 10, "hi", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SanitizeAndValidate(c.in, c.minLen, c.maxLen)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if err == nil && got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
