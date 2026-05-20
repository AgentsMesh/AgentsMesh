package slugkit

import (
	"context"
	"fmt"
	"strings"
)

type UniquenessChecker func(ctx context.Context, candidate string) (available bool, err error)

func FromExistsCheck(exists func(context.Context, string) (bool, error)) UniquenessChecker {
	return func(ctx context.Context, candidate string) (bool, error) {
		e, err := exists(ctx, candidate)
		if err != nil {
			return false, err
		}
		return !e, nil
	}
}

const (
	generatorMaxAttempts = 100
	suffixReserve = 4
)

func GenerateUnique(ctx context.Context, raw string, check UniquenessChecker) (string, error) {
	base, err := SanitizeAndValidate(raw)
	if err != nil {
		return "", err
	}
	if len(base) > MaxLen-suffixReserve {
		truncated := strings.TrimRight(base[:MaxLen-suffixReserve], "-")
		if err := Validate(truncated); err != nil {
			return "", fmt.Errorf("base %q too long to suffix safely (truncation invalid): %w", base, err)
		}
		base = truncated
	}

	for i := 0; i < generatorMaxAttempts; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		if err := Validate(candidate); err != nil {
			continue
		}
		ok, err := check(ctx, candidate)
		if err != nil {
			return "", err
		}
		if ok {
			return candidate, nil
		}
	}
	return "", ErrCollisionExhausted
}

// TrySeeds walks the seed list in order, returning the first that
// GenerateUnique can place. Empty seeds are skipped. Returns ("", false)
// when every seed exhausts retries — callers apply their own fallback
// (random handle, hard error, etc.).
func TrySeeds(ctx context.Context, seeds []string, check UniquenessChecker) (string, bool) {
	for _, seed := range seeds {
		if seed == "" {
			continue
		}
		if s, err := GenerateUnique(ctx, seed, check); err == nil {
			return s, true
		}
	}
	return "", false
}
