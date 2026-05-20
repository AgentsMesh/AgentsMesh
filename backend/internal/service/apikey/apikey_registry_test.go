package apikey

import (
	"context"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

func TestEnsureUniqueSlug_SanitizesName(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewAPIKeyRepository(db), nil)
	ctx := context.Background()

	slug, err := svc.EnsureUniqueSlug(ctx, 1, "Production API")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if slug != "production-api" {
		t.Errorf("got %q, want production-api", slug)
	}
	if err := slugkit.Validate(slug); err != nil {
		t.Errorf("slug fails Validate: %v", err)
	}
}

func TestEnsureUniqueSlug_CollisionSuffix(t *testing.T) {
	db := testkit.SetupTestDB(t)
	svc := NewService(infra.NewAPIKeyRepository(db), nil)
	ctx := context.Background()

	first, err := svc.CreateAPIKey(ctx, &CreateAPIKeyRequest{
		OrganizationID: 1, Name: "Production", Scopes: []string{"pods:read"}, CreatedBy: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.APIKey.Slug == nil || *first.APIKey.Slug != "production" {
		t.Errorf("first key should get slug=production, got %v", first.APIKey.Slug)
	}

	slug, err := svc.EnsureUniqueSlug(ctx, 1, "Production")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(slug, "production-") || slug == "production" {
		t.Errorf("expected collision suffix, got %q", slug)
	}
}
