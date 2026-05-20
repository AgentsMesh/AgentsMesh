package channel

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

func TestEnsureUniqueSlug_SanitizesName(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	slug, err := svc.EnsureUniqueSlug(ctx, 1, "My Channel")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if slug != "my-channel" {
		t.Errorf("got %q, want my-channel", slug)
	}
	if err := slugkit.Validate(slug); err != nil {
		t.Errorf("slug fails Validate: %v", err)
	}
}

func TestEnsureUniqueSlug_CollisionSuffix(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	if _, err := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "Engineering"}); err != nil {
		t.Fatal(err)
	}
	slug, err := svc.EnsureUniqueSlug(ctx, 1, "Engineering")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(slug, "engineering-") || slug == "engineering" {
		t.Errorf("expected collision suffix, got %q", slug)
	}
}

func TestEnsureUniqueSlug_UnicodeNameFallsBack(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	// Unicode-only names sanitize to empty; GenerateUnique returns error.
	_, err := svc.EnsureUniqueSlug(ctx, 1, "我的频道")
	if err == nil {
		t.Fatal("expected error for unicode-only name (CreateChannel itself handles via fallback)")
	}
}

func TestCreateChannel_DerivesSlugFromName(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1,
		Name:           "Engineering",
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if ch.Slug == nil || *ch.Slug != "engineering" {
		t.Errorf("expected slug=engineering, got %v", ch.Slug)
	}
}

func TestCreateChannel_DisplaykitSanitizesName(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	// Zero-width space () and BOM () embedded as escapes —
	// Go source files cannot contain literal U+FEFF mid-file.
	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1,
		Name:           "  ali\u200bce-room\ufeff  ",
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if ch.Name != "alice-room" {
		t.Errorf("expected sanitized name 'alice-room', got %q", ch.Name)
	}
}
