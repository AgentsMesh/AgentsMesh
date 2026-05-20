package apikey

import "testing"

func TestAPIKey_ValidateIdentifiers_NilSlugPasses(t *testing.T) {
	k := &APIKey{Name: "Test", Slug: nil}
	if err := k.ValidateIdentifiers(); err != nil {
		t.Errorf("nullable slug should pass: %v", err)
	}
}

func TestAPIKey_ValidateIdentifiers_RejectsInvalidSlug(t *testing.T) {
	bad := "API.Key"
	k := &APIKey{Name: "Test", Slug: &bad}
	if err := k.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject slug with dot")
	}
}

func TestAPIKey_ValidateIdentifiers_AcceptsValidSlug(t *testing.T) {
	good := "production"
	k := &APIKey{Name: "Test", Slug: &good}
	if err := k.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid slug: %v", err)
	}
}
