package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

// RelayClaims represents JWT claims for relay token
// Note: SessionID has been removed - channels are now identified by PodKey only
type RelayClaims struct {
	PodKey   string `json:"pod_key"`
	RunnerID int64  `json:"runner_id"`
	UserID   int64  `json:"user_id"` // 0 for runner tokens
	OrgID    int64  `json:"org_id"`
	jwt.RegisteredClaims
}

// IsRunnerToken returns true if this is a runner-issued token (UserID == 0).
func (c *RelayClaims) IsRunnerToken() bool { return c.UserID == 0 }

// IsBrowserToken returns true if this is a browser-issued token (UserID != 0).
func (c *RelayClaims) IsBrowserToken() bool { return c.UserID != 0 }

// TokenValidator validates relay tokens
type TokenValidator struct {
	secretKey []byte
	issuer    string
}

// NewTokenValidator creates a new token validator.
// Panics if secret is empty to prevent validating tokens with a zero-length HMAC key.
func NewTokenValidator(secret, issuer string) *TokenValidator {
	if secret == "" {
		panic("relay token validator secret must not be empty")
	}
	return &TokenValidator{
		secretKey: []byte(secret),
		issuer:    issuer,
	}
}

// ValidateToken validates a relay token and returns claims
func (v *TokenValidator) ValidateToken(tokenString string) (*RelayClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RelayClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return v.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*RelayClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify issuer if configured
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateToken generates a relay token (used by Backend)
// Note: SessionID parameter has been removed - channels are identified by PodKey only
func GenerateToken(secret, issuer, podKey string, runnerID, userID, orgID int64, expiry time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(expiry)

	claims := &RelayClaims{
		PodKey:   podKey,
		RunnerID: runnerID,
		UserID:   userID,
		OrgID:    orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    issuer,
			Subject:   podKey,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
