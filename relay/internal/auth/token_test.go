package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret = "test-secret-key-for-testing"
	testIssuer = "test-issuer"
)

func TestNewTokenValidator(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	if v == nil || string(v.secretKey) != testSecret || v.issuer != testIssuer {
		t.Error("NewTokenValidator failed")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(testSecret, testIssuer, "pod-1", 1, 2, 3, time.Hour)
	if err != nil || token == "" {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	v := NewTokenValidator(testSecret, testIssuer)
	claims, err := v.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.PodKey != "pod-1" ||
		claims.RunnerID != 1 || claims.UserID != 2 || claims.OrgID != 3 ||
		claims.Issuer != testIssuer || claims.Subject != "pod-1" {
		t.Error("claims mismatch")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	token, _ := GenerateToken(testSecret, testIssuer, "pod-1", 1, 2, 3, time.Hour)
	if claims, err := v.ValidateToken(token); err != nil || claims == nil {
		t.Errorf("ValidateToken failed: %v", err)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	token, _ := GenerateToken(testSecret, testIssuer, "pod-1", 1, 2, 3, -time.Hour)
	if _, err := v.ValidateToken(token); err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	token, _ := GenerateToken("wrong-secret", testIssuer, "pod-1", 1, 2, 3, time.Hour)
	if _, err := v.ValidateToken(token); err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_InvalidIssuer(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	token, _ := GenerateToken(testSecret, "wrong-issuer", "pod-1", 1, 2, 3, time.Hour)
	if _, err := v.ValidateToken(token); err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_NoIssuerCheck(t *testing.T) {
	v := NewTokenValidator(testSecret, "")
	token, _ := GenerateToken(testSecret, "any-issuer", "pod-1", 1, 2, 3, time.Hour)
	claims, err := v.ValidateToken(token)
	if err != nil || claims.Issuer != "any-issuer" {
		t.Error("should succeed when issuer check disabled")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	for _, token := range []string{"", "not-a-token", "eyJhbGciOiJIUzI1NiJ9", "eyJ.!!!.xyz"} {
		if _, err := v.ValidateToken(token); err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken for %q", token)
		}
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	v := NewTokenValidator(testSecret, testIssuer)
	claims := &RelayClaims{PodKey: "pod-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()), Issuer: testIssuer}}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := v.ValidateToken(tokenString); err != ErrInvalidToken {
		t.Error("expected ErrInvalidToken for wrong signing method")
	}
}

func TestRelayClaims_AllFields(t *testing.T) {
	now := time.Now()
	claims := &RelayClaims{PodKey: "p1", RunnerID: 100, UserID: 200, OrgID: 300,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), IssuedAt: jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now), Issuer: testIssuer, Subject: "p1"}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))
	v := NewTokenValidator(testSecret, testIssuer)
	decoded, err := v.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if decoded.PodKey != "p1" ||
		decoded.RunnerID != 100 || decoded.UserID != 200 || decoded.OrgID != 300 ||
		decoded.Issuer != testIssuer || decoded.Subject != "p1" {
		t.Error("decoded claims mismatch")
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrInvalidToken.Error() != "invalid token" {
		t.Error("ErrInvalidToken message wrong")
	}
	if ErrTokenExpired.Error() != "token expired" {
		t.Error("ErrTokenExpired message wrong")
	}
}
