//go:build !windows

package terminal

import (
	"errors"
	"io"
	"syscall"
	"testing"
)

func TestNormalizePTYReadError(t *testing.T) {
	if err := normalizePTYReadError(syscall.EIO); !errors.Is(err, io.EOF) {
		t.Fatalf("EIO normalized to %v, want EOF", err)
	}
	want := errors.New("read failed")
	if err := normalizePTYReadError(want); !errors.Is(err, want) {
		t.Fatalf("ordinary error normalized to %v, want %v", err, want)
	}
	if err := normalizePTYReadError(nil); err != nil {
		t.Fatalf("nil read error normalized to %v", err)
	}
}
