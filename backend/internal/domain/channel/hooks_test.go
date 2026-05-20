package channel

import "testing"

func TestChannel_ValidateIdentifiers_NilSlugPasses(t *testing.T) {
	c := &Channel{Name: "Test", Slug: nil}
	if err := c.ValidateIdentifiers(); err != nil {
		t.Errorf("nullable slug should pass: %v", err)
	}
}

func TestChannel_ValidateIdentifiers_RejectsInvalidSlug(t *testing.T) {
	bad := "Foo.Bar"
	c := &Channel{Name: "Test", Slug: &bad}
	if err := c.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject slug with dot")
	}
}

func TestChannel_ValidateIdentifiers_AcceptsValidSlug(t *testing.T) {
	good := "my-channel"
	c := &Channel{Name: "Test", Slug: &good}
	if err := c.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid slug: %v", err)
	}
}
