package clipboard

import (
	"fmt"
	"os"
	"path/filepath"
)

const shimDirName = ".clipboard-shim"

// ShimBinDir returns the path to the clipboard shim bin directory.
func ShimBinDir(sandboxRoot string) string {
	return filepath.Join(sandboxRoot, shimDirName, "bin")
}

// dataDir returns the path to the clipboard shim data directory.
func dataDir(sandboxRoot string) string {
	return filepath.Join(sandboxRoot, shimDirName, "data")
}

// SetupShims creates clipboard shim scripts in the sandbox.
// The shims intercept xclip and osascript calls to serve images from local storage.
func SetupShims(sandboxRoot string) error {
	if sandboxRoot == "" {
		return fmt.Errorf("sandbox root is empty")
	}

	binDir := ShimBinDir(sandboxRoot)
	dDir := dataDir(sandboxRoot)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("create shim bin dir: %w", err)
	}
	if err := os.MkdirAll(dDir, 0755); err != nil {
		return fmt.Errorf("create shim data dir: %w", err)
	}

	// Write xclip shim
	if err := os.WriteFile(filepath.Join(binDir, "xclip"), []byte(xclipShim), 0755); err != nil {
		return fmt.Errorf("write xclip shim: %w", err)
	}

	// Write osascript shim
	if err := os.WriteFile(filepath.Join(binDir, "osascript"), []byte(osascriptShim), 0755); err != nil {
		return fmt.Errorf("write osascript shim: %w", err)
	}

	return nil
}

// WriteImage writes image data to the clipboard shim storage.
// The shim scripts only serve image/png (xclip shim checks for "image/png",
// osascript shim returns «class PNGf»), so this backend is inherently PNG-only.
func WriteImage(sandboxRoot string, mimeType string, data []byte) error {
	if mimeType != "image/png" {
		return fmt.Errorf("shim clipboard only supports image/png, got %s", mimeType)
	}
	dDir := dataDir(sandboxRoot)
	if err := os.MkdirAll(dDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	imagePath := filepath.Join(dDir, "image.png")
	if err := os.WriteFile(imagePath, data, 0644); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	return nil
}

// xclip shim script for Linux / Claude Code on Linux
const xclipShim = `#!/bin/bash
SHIM_DIR="$(dirname "$(dirname "$0")")"/data
# Detect if this is a read (output) operation with image/png target
if [[ "$*" == *"-o"* ]] || [[ "$*" == *"--output"* ]]; then
  if [[ "$*" == *"image/png"* ]]; then
    if [ -f "$SHIM_DIR/image.png" ]; then
      cat "$SHIM_DIR/image.png"
      exit 0
    fi
  fi
  # Check for target listing
  if [[ "$*" == *"-t"* ]] && [[ "$*" != *"image/"* ]]; then
    if [ -f "$SHIM_DIR/image.png" ]; then
      echo "image/png"
      exit 0
    fi
  fi
fi
# For all other operations, try real xclip
REAL_XCLIP=$(command -v xclip 2>/dev/null | grep -v "$0" | head -1)
if [ -n "$REAL_XCLIP" ]; then
  exec "$REAL_XCLIP" "$@"
fi
exit 1
`

// osascript shim script for macOS / Claude Code on macOS
const osascriptShim = `#!/bin/bash
SHIM_DIR="$(dirname "$(dirname "$0")")"/data
# Detect clipboard image read: osascript -e 'the clipboard as «class PNGf»'
if [[ "$*" == *"clipboard"* ]] && [[ "$*" == *"PNGf"* ]]; then
  if [ -f "$SHIM_DIR/image.png" ]; then
    # Return hex-encoded PNG data in AppleScript format
    echo -n "«data PNGf"
    xxd -p "$SHIM_DIR/image.png" | tr -d '\n'
    echo "»"
    exit 0
  fi
fi
# For all other operations, use real osascript
REAL_OSASCRIPT="/usr/bin/osascript"
if [ -x "$REAL_OSASCRIPT" ]; then
  exec "$REAL_OSASCRIPT" "$@"
fi
exit 1
`

// ShimBackend implements Backend for headless environments.
// It creates interceptor scripts that serve stored images when agents
// call xclip/osascript to read the clipboard.
type ShimBackend struct{}

func (b *ShimBackend) Name() string {
	return "shim"
}

func (b *ShimBackend) Setup(sandboxRoot string) error {
	return SetupShims(sandboxRoot)
}

func (b *ShimBackend) WriteImage(sandboxRoot string, mimeType string, data []byte) error {
	return WriteImage(sandboxRoot, mimeType, data)
}

// EnvOverrides returns env var overrides for the pod process.
// For PATH, returns just the shim bin directory to prepend.
// The caller is responsible for merging with the existing PATH.
func (b *ShimBackend) EnvOverrides(sandboxRoot string) map[string]string {
	if sandboxRoot == "" {
		return nil
	}
	shimBinDir := ShimBinDir(sandboxRoot)
	if _, err := os.Stat(shimBinDir); err != nil {
		return nil
	}
	return map[string]string{
		"PATH": shimBinDir,
	}
}
