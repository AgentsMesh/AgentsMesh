package user

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

func TestEnsureUniqueUsername(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	cases := []struct {
		name      string
		seeds     []string
		wantMatch string
	}{
		{"clean_seed_used_as_is", []string{"alice"}, "^alice$"},
		{"sanitize_dot", []string{"kudin.private"}, "^kudin-private$"},
		{"sanitize_email_localpart", []string{"foo+bar"}, "^foo-bar$"},
		{"sanitize_uppercase", []string{"John.Doe"}, "^john-doe$"},
		{"empty_seed_skipped", []string{"", "alice"}, "^alice$"},
		{"unicode_seed_falls_back", []string{"用户名"}, "^user-[0-9a-f]{8}$"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.EnsureUniqueUsername(ctx, c.seeds)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if err := slugkit.Validate(got); err != nil {
				t.Errorf("result %q fails Validate: %v", got, err)
			}
			if !regexp.MustCompile(c.wantMatch).MatchString(got) {
				t.Errorf("got %q, want match %s", got, c.wantMatch)
			}
		})
	}
}

func TestEnsureUniqueUsername_CollisionSuffix(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	if _, err := svc.Create(ctx, &CreateRequest{Email: "a@x.com", Username: "alice"}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.EnsureUniqueUsername(ctx, []string{"alice"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "alice-") || got == "alice" {
		t.Errorf("expected collision suffix on 'alice', got %q", got)
	}
	if err := slugkit.Validate(got); err != nil {
		t.Errorf("collision result %q fails Validate: %v", got, err)
	}
}

func TestEnsureUniqueUsername_AllSeedsEmpty(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	got, err := svc.EnsureUniqueUsername(ctx, []string{"", "", ""})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile("^user-[0-9a-f]{8}$").MatchString(got) {
		t.Errorf("expected random fallback, got %q", got)
	}
}

// Mirrors the bug that triggered this refactor: kudin.private@gmail.com →
// previously stored "kudin.private", causing personal workspace slug to 422.
func TestEnsureUniqueUsername_KudinPrivateBugRegression(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(infra.NewUserRepository(db))
	ctx := context.Background()

	emailLocal := "kudin.private"
	got, err := svc.EnsureUniqueUsername(ctx, []string{emailLocal})
	if err != nil {
		t.Fatal(err)
	}
	if got != "kudin-private" {
		t.Errorf("got %q, want kudin-private", got)
	}
	if err := slugkit.Validate(got); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}
