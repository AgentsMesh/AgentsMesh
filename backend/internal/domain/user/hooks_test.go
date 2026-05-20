package user

import "testing"

func TestUser_ValidateIdentifiers_RejectsInvalidUsername(t *testing.T) {
	u := &User{Username: "kudin.private"}
	if err := u.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject username with dot (kudin.private regression)")
	}
}

func TestUser_ValidateIdentifiers_AcceptsValidUsername(t *testing.T) {
	u := &User{Username: "kudin-private"}
	if err := u.ValidateIdentifiers(); err != nil {
		t.Errorf("validator rejected valid username: %v", err)
	}
}

func TestUser_ValidateIdentifiers_RejectsTooShort(t *testing.T) {
	u := &User{Username: "x"}
	if err := u.ValidateIdentifiers(); err == nil {
		t.Fatal("validator should reject single-char username")
	}
}
