package organization

import "testing"

func TestOrganization_ValidateIdentifiers_RejectsInvalidSlug(t *testing.T) {
	o := &Organization{Slug: "Foo.Bar"}
	if err := o.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject slug with dot")
	}
}

func TestOrganization_ValidateIdentifiers_AcceptsValidSlug(t *testing.T) {
	o := &Organization{Slug: "valid-slug"}
	if err := o.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid slug: %v", err)
	}
}
