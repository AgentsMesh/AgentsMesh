package slugkit

import (
	"errors"
	"testing"
)

func TestValidateIdentifier_EmptyPasses(t *testing.T) {
	if err := ValidateIdentifier("channels.slug", ""); err != nil {
		t.Errorf("empty should pass (nullable field), got %v", err)
	}
}

func TestValidateIdentifier_Valid(t *testing.T) {
	if err := ValidateIdentifier("channels.slug", "my-channel"); err != nil {
		t.Errorf("valid slug should pass, got %v", err)
	}
}

func TestValidateIdentifier_InvalidIncludesFieldName(t *testing.T) {
	err := ValidateIdentifier("users.username", "kudin.private")
	if err == nil {
		t.Fatal("expected error for slug with dot")
	}
	if !errors.Is(err, ErrInvalidFormat) {
		t.Errorf("expected ErrInvalidFormat wrapped, got %v", err)
	}
	if msg := err.Error(); msg[:14] != "users.username" {
		t.Errorf("error should be prefixed with field name, got %q", msg)
	}
}

func TestValidateIdentifier_TooShort(t *testing.T) {
	err := ValidateIdentifier("channels.slug", "x")
	if !errors.Is(err, ErrTooShort) {
		t.Errorf("expected ErrTooShort, got %v", err)
	}
}

func TestValidateIdentifier_Reserved(t *testing.T) {
	err := ValidateIdentifier("organizations.slug", "admin")
	if !errors.Is(err, ErrReserved) {
		t.Errorf("expected ErrReserved, got %v", err)
	}
}
