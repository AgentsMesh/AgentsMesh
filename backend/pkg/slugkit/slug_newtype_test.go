package slugkit

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNewFromTrusted(t *testing.T) {
	cases := []struct {
		name, in string
		wantErr  error
	}{
		{"valid", "kudin-private", nil},
		{"valid_single_segment", "alice", nil},
		{"valid_with_digits", "foo-123", nil},
		{"empty", "", ErrEmpty},
		{"too_short", "a", ErrTooShort},
		{"uppercase", "Foo", ErrInvalidFormat},
		{"dot", "foo.bar", ErrInvalidFormat},
		{"underscore", "foo_bar", ErrInvalidFormat},
		{"leading_hyphen", "-foo", ErrInvalidFormat},
		{"trailing_hyphen", "foo-", ErrInvalidFormat},
		{"double_hyphen", "foo--bar", ErrInvalidFormat},
		{"reserved", "admin", ErrReserved},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sl, err := NewFromTrusted(c.in)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("NewFromTrusted(%q): err = %v, want %v", c.in, err, c.wantErr)
			}
			if err == nil && sl.String() != c.in {
				t.Errorf("roundtrip: got %q, want %q", sl.String(), c.in)
			}
		})
	}
}

func TestMustNewForTest_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid input")
		}
	}()
	MustNewForTest("foo.bar")
}

func TestMustNewForTest_Valid(t *testing.T) {
	if got := MustNewForTest("foo-bar").String(); got != "foo-bar" {
		t.Fatalf("got %q", got)
	}
}

func TestSlug_Scan(t *testing.T) {
	cases := []struct {
		name    string
		in      interface{}
		want    Slug
		wantErr bool
	}{
		{"string", "foo-bar", "foo-bar", false},
		{"bytes", []byte("foo-bar"), "foo-bar", false},
		{"nil", nil, "", false},
		{"int", 42, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var s Slug
			err := s.Scan(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("Scan(%v): err = %v, wantErr = %v", c.in, err, c.wantErr)
			}
			if err == nil && s != c.want {
				t.Errorf("got %q, want %q", s, c.want)
			}
		})
	}
}

// Scan must accept old/invalid rows during migration phases (CHECK NOT VALID).
func TestSlug_Scan_BypassesValidate(t *testing.T) {
	var s Slug
	if err := s.Scan("legacy.invalid"); err != nil {
		t.Fatalf("Scan should bypass Validate, got %v", err)
	}
	if s != "legacy.invalid" {
		t.Errorf("got %q", s)
	}
}

func TestSlug_Value(t *testing.T) {
	v, err := Slug("foo-bar").Value()
	if err != nil {
		t.Fatal(err)
	}
	if v != "foo-bar" {
		t.Errorf("got %v", v)
	}
}

func TestSlug_MarshalJSON(t *testing.T) {
	b, err := json.Marshal(Slug("foo-bar"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"foo-bar"` {
		t.Errorf("got %s", b)
	}
}

func TestSlug_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name, in string
		want     Slug
		wantErr  bool
	}{
		{"valid", `"foo-bar"`, "foo-bar", false},
		{"invalid_dot", `"foo.bar"`, "", true},
		{"empty", `""`, "", true},
		{"non_string", `42`, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var s Slug
			err := json.Unmarshal([]byte(c.in), &s)
			if (err != nil) != c.wantErr {
				t.Fatalf("Unmarshal(%s): err = %v, wantErr = %v", c.in, err, c.wantErr)
			}
			if err == nil && s != c.want {
				t.Errorf("got %q, want %q", s, c.want)
			}
		})
	}
}
