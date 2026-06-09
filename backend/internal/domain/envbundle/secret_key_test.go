package envbundle

import "testing"

func TestIsNonSecretKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"ANTHROPIC_BASE_URL", true},
		{"ANTHROPIC_API_KEY", false},
		{"ANTHROPIC_AUTH_TOKEN", false},
		{"anthropic_base_url", false}, // exact match, case-sensitive
		{"UNKNOWN_KEY", false},        // default-deny
		{"", false},
	}
	for _, c := range cases {
		if got := IsNonSecretKey(c.key); got != c.want {
			t.Errorf("IsNonSecretKey(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}
