package infra

import (
	"errors"

	"gorm.io/gorm"
)

// isNotFound returns true when err is gorm.ErrRecordNotFound.
// Centralises the check so each repository file can use a short helper.
func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
