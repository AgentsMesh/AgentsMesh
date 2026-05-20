package slugkit

import "testing"

func TestSanitizeDNS(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"basic", "John.Doe", "john-doe"},
		{"underscore", "user_123", "user-123"},
		{"trailing_dot", "name.", "name"},
		{"empty_input", "", ""},
		{"only_punct", "...", ""},
		{"unicode_stripped", "name用户", "name"},
		{
			name: "truncate_to_63",
			in:   "a234567890b234567890c234567890d234567890e234567890f234567890g234extra",
			want: "a234567890b234567890c234567890d234567890e234567890f234567890g23",
		},
		{
			name: "truncate_trims_trailing_hyphen",
			in:   "a234567890b234567890c234567890d234567890e234567890f234567890g2-extra",
			want: "a234567890b234567890c234567890d234567890e234567890f234567890g2",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SanitizeDNS(c.in)
			if got != c.want {
				t.Errorf("SanitizeDNS(%q) = %q, want %q", c.in, got, c.want)
			}
			if len(got) > DNSMaxLabelLen {
				t.Errorf("output too long: %d", len(got))
			}
		})
	}
}
