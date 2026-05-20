package user

import (
	"reflect"
	"testing"
)

// usernameSeeds priority: provider username > email local-part > human name.
// Empty/whitespace seeds drop silently. Order matters because the OAuth
// funnel feeds these to EnsureUniqueUsername in this exact sequence.

func TestUsernameSeeds_Priority(t *testing.T) {
	cases := []struct {
		name, provider, email, full string
		want                        []string
	}{
		{"all_three", "octo", "octo@gh.com", "Octo Cat", []string{"octo", "octo", "Octo Cat"}},
		{"no_provider", "", "kudin.private@gmail.com", "Roman Kudin", []string{"kudin.private", "Roman Kudin"}},
		{"only_email", "", "alice@x.com", "", []string{"alice"}},
		{"only_provider", "bob", "", "", []string{"bob"}},
		{"only_name", "", "", "John Doe", []string{"John Doe"}},
		{"all_empty", "", "", "", []string{}},
		{"email_no_at", "", "weird", "", []string{"weird"}},
		{"email_empty_local", "", "@nolocal.com", "", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := usernameSeeds(c.provider, c.email, c.full)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("usernameSeeds(%q,%q,%q) = %v, want %v",
					c.provider, c.email, c.full, got, c.want)
			}
		})
	}
}
