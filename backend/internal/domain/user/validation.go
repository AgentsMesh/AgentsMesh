package user

import (
	"errors"
	"regexp"
)

const (
	UsernameMinLen = 3
	UsernameMaxLen = 50
)

var ErrInvalidUsername = errors.New("username must contain only letters, numbers, hyphens, and underscores, length 3-50")

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func ValidateUsername(s string) error {
	if len(s) < UsernameMinLen || len(s) > UsernameMaxLen {
		return ErrInvalidUsername
	}
	if !usernamePattern.MatchString(s) {
		return ErrInvalidUsername
	}
	return nil
}
