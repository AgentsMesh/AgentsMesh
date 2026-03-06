package clipboard

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Backend abstracts clipboard operations for different environments.
// Native backends write to the real system clipboard; the shim backend
// creates interceptor scripts for headless environments.
type Backend interface {
	// Name returns the backend name for logging (e.g. "native:xclip", "shim").
	Name() string
	// Setup prepares the backend for a pod sandbox (shim creates scripts, native is no-op).
	Setup(sandboxRoot string) error
	// WriteImage writes image data to the clipboard.
	// Currently only "image/png" is fully supported across all backends.
	// Returns an error if the mimeType is not supported by the backend.
	WriteImage(sandboxRoot string, mimeType string, data []byte) error
	// EnvOverrides returns env var overrides for the pod process.
	// For PATH, returns just the directory to prepend (not the full PATH).
	// The caller merges with the existing PATH. Native returns nil.
	EnvOverrides(sandboxRoot string) map[string]string
}

// detectEnv holds environment queries used by detection logic.
// Extracted for testability (runtime.GOOS is a compile-time constant).
type detectEnv struct {
	goos     string
	getenv   func(string) string
	lookPath func(string) (string, error)
}

// detect implements the backend selection logic with injectable dependencies.
func detect(env detectEnv) Backend {
	// 0. Windows: no Unix clipboard tools available, use no-op backend.
	//    Native Windows clipboard support (clip.exe / PowerShell Set-Clipboard)
	//    can be added in the future if needed.
	if env.goos == "windows" {
		return &NoopBackend{}
	}
	// 1. macOS: pasteboard is always available (no display server needed)
	if env.goos == "darwin" {
		if _, err := env.lookPath("osascript"); err == nil {
			return &NativeBackend{tool: "osascript"}
		}
	}
	// 2. Wayland
	if env.getenv("WAYLAND_DISPLAY") != "" {
		if _, err := env.lookPath("wl-copy"); err == nil {
			return &NativeBackend{tool: "wl-copy"}
		}
	}
	// 3. X11
	if env.getenv("DISPLAY") != "" {
		if _, err := env.lookPath("xclip"); err == nil {
			return &NativeBackend{tool: "xclip"}
		}
	}
	// 4. Headless fallback (Unix bash shim scripts)
	return &ShimBackend{}
}

// Detect returns the best available clipboard backend for the current environment.
func Detect() Backend {
	return detect(detectEnv{
		goos:     runtime.GOOS,
		getenv:   os.Getenv,
		lookPath: exec.LookPath,
	})
}

// NativeBackend writes images to the real system clipboard using native tools.
type NativeBackend struct {
	tool string // "xclip", "wl-copy", or "osascript"
}

func (b *NativeBackend) Name() string {
	return "native:" + b.tool
}

func (b *NativeBackend) Setup(sandboxRoot string) error {
	// Native clipboard doesn't need sandbox setup
	return nil
}

func (b *NativeBackend) WriteImage(sandboxRoot string, mimeType string, data []byte) error {
	switch b.tool {
	case "xclip":
		cmd := exec.Command("xclip", "-selection", "clipboard", "-t", mimeType, "-i")
		cmd.Stdin = bytes.NewReader(data)
		return cmd.Run()
	case "wl-copy":
		cmd := exec.Command("wl-copy", "--type", mimeType)
		cmd.Stdin = bytes.NewReader(data)
		return cmd.Run()
	case "osascript":
		if mimeType != "image/png" {
			return fmt.Errorf("osascript clipboard only supports image/png, got %s", mimeType)
		}
		tmp, err := os.CreateTemp("", "clipboard-*.png")
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.Write(data); err != nil {
			tmp.Close()
			return fmt.Errorf("write temp file: %w", err)
		}
		tmp.Close()
		return exec.Command("osascript", "-e",
			fmt.Sprintf(`set the clipboard to (read (POSIX file "%s") as «class PNGf»)`, tmp.Name())).Run()
	}
	return fmt.Errorf("unsupported tool: %s", b.tool)
}

func (b *NativeBackend) EnvOverrides(sandboxRoot string) map[string]string {
	// Native clipboard doesn't need PATH overrides
	return nil
}

// NoopBackend is a clipboard backend that does nothing.
// Used on platforms where clipboard shim scripts are not applicable (e.g. Windows).
type NoopBackend struct{}

func (b *NoopBackend) Name() string                        { return "noop" }
func (b *NoopBackend) Setup(string) error                  { return nil }
func (b *NoopBackend) WriteImage(string, string, []byte) error { return fmt.Errorf("clipboard not supported on this platform") }
func (b *NoopBackend) EnvOverrides(string) map[string]string   { return nil }
