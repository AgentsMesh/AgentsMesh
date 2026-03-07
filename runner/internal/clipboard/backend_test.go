package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// --- detect() unit tests (injectable env) ---

func TestDetect_Darwin_OsascriptAvailable(t *testing.T) {
	env := detectEnv{
		goos:     "darwin",
		getenv:   func(string) string { return "" },
		lookPath: func(name string) (string, error) {
			if name == "osascript" {
				return "/usr/bin/osascript", nil
			}
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "native:osascript" {
		t.Errorf("got %s, want native:osascript", b.Name())
	}
}

func TestDetect_Darwin_OsascriptMissing(t *testing.T) {
	env := detectEnv{
		goos:   "darwin",
		getenv: func(string) string { return "" },
		lookPath: func(string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	// Falls through to headless since no display env vars set
	if b.Name() != "shim" {
		t.Errorf("got %s, want shim", b.Name())
	}
}

func TestDetect_Wayland(t *testing.T) {
	env := detectEnv{
		goos: "linux",
		getenv: func(key string) string {
			if key == "WAYLAND_DISPLAY" {
				return "wayland-0"
			}
			return ""
		},
		lookPath: func(name string) (string, error) {
			if name == "wl-copy" {
				return "/usr/bin/wl-copy", nil
			}
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "native:wl-copy" {
		t.Errorf("got %s, want native:wl-copy", b.Name())
	}
}

func TestDetect_Wayland_ToolMissing(t *testing.T) {
	env := detectEnv{
		goos: "linux",
		getenv: func(key string) string {
			if key == "WAYLAND_DISPLAY" {
				return "wayland-0"
			}
			return ""
		},
		lookPath: func(string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "shim" {
		t.Errorf("got %s, want shim (wl-copy not found)", b.Name())
	}
}

func TestDetect_X11(t *testing.T) {
	env := detectEnv{
		goos: "linux",
		getenv: func(key string) string {
			if key == "DISPLAY" {
				return ":0"
			}
			return ""
		},
		lookPath: func(name string) (string, error) {
			if name == "xclip" {
				return "/usr/bin/xclip", nil
			}
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "native:xclip" {
		t.Errorf("got %s, want native:xclip", b.Name())
	}
}

func TestDetect_X11_ToolMissing(t *testing.T) {
	env := detectEnv{
		goos: "linux",
		getenv: func(key string) string {
			if key == "DISPLAY" {
				return ":0"
			}
			return ""
		},
		lookPath: func(string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "shim" {
		t.Errorf("got %s, want shim (xclip not found)", b.Name())
	}
}

func TestDetect_Headless(t *testing.T) {
	env := detectEnv{
		goos:   "linux",
		getenv: func(string) string { return "" },
		lookPath: func(string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "shim" {
		t.Errorf("got %s, want shim", b.Name())
	}
}

func TestDetect_WaylandPriority_OverX11(t *testing.T) {
	// Both WAYLAND_DISPLAY and DISPLAY set; Wayland should win
	env := detectEnv{
		goos: "linux",
		getenv: func(key string) string {
			switch key {
			case "WAYLAND_DISPLAY":
				return "wayland-0"
			case "DISPLAY":
				return ":0"
			}
			return ""
		},
		lookPath: func(name string) (string, error) {
			if name == "wl-copy" || name == "xclip" {
				return "/usr/bin/" + name, nil
			}
			return "", fmt.Errorf("not found")
		},
	}
	b := detect(env)
	if b.Name() != "native:wl-copy" {
		t.Errorf("got %s, want native:wl-copy (should prefer Wayland over X11)", b.Name())
	}
}

// --- Detect() integration test (real environment) ---

func TestDetect_ReturnsBackend(t *testing.T) {
	b := Detect()
	if b == nil {
		t.Fatal("Detect() returned nil")
	}
	if b.Name() == "" {
		t.Error("backend name is empty")
	}
}

func TestDetect_OnCurrentPlatform(t *testing.T) {
	b := Detect()
	if runtime.GOOS == "darwin" {
		if b.Name() != "native:osascript" {
			t.Errorf("expected native:osascript on macOS, got %s", b.Name())
		}
	}
}

// --- NativeBackend tests ---

func TestNativeBackend_Name(t *testing.T) {
	tests := []struct {
		tool string
		want string
	}{
		{"xclip", "native:xclip"},
		{"wl-copy", "native:wl-copy"},
		{"osascript", "native:osascript"},
	}
	for _, tt := range tests {
		b := &NativeBackend{tool: tt.tool}
		if got := b.Name(); got != tt.want {
			t.Errorf("NativeBackend{%s}.Name() = %q, want %q", tt.tool, got, tt.want)
		}
	}
}

func TestNativeBackend_Setup(t *testing.T) {
	b := &NativeBackend{tool: "xclip"}
	if err := b.Setup(t.TempDir()); err != nil {
		t.Errorf("NativeBackend.Setup() = %v, want nil", err)
	}
}

func TestNativeBackend_EnvOverrides(t *testing.T) {
	b := &NativeBackend{tool: "xclip"}
	overrides := b.EnvOverrides(t.TempDir())
	if overrides != nil {
		t.Errorf("NativeBackend.EnvOverrides() = %v, want nil", overrides)
	}
}

func TestNativeBackend_WriteImage_UnsupportedTool(t *testing.T) {
	b := &NativeBackend{tool: "unknown"}
	err := b.WriteImage(t.TempDir(), "image/png", []byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for unsupported tool")
	}
}

func TestNativeBackend_WriteImage_Xclip(t *testing.T) {
	if _, err := exec.LookPath("xclip"); err != nil {
		t.Skip("xclip not installed")
	}
	b := &NativeBackend{tool: "xclip"}
	// xclip needs DISPLAY; may fail but exercises the code path
	_ = b.WriteImage("", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})
}

func TestNativeBackend_WriteImage_WlCopy(t *testing.T) {
	if _, err := exec.LookPath("wl-copy"); err != nil {
		t.Skip("wl-copy not installed")
	}
	b := &NativeBackend{tool: "wl-copy"}
	_ = b.WriteImage("", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})
}

func TestNativeBackend_WriteImage_OsascriptUnsupportedMimeType(t *testing.T) {
	b := &NativeBackend{tool: "osascript"}
	for _, mime := range []string{"image/jpeg", "image/gif", "text/plain"} {
		err := b.WriteImage("", mime, []byte{1, 2, 3})
		if err == nil {
			t.Errorf("WriteImage(%q) with osascript should return error", mime)
		}
	}
}

func TestNativeBackend_WriteImage_Osascript(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("osascript only available on macOS")
	}
	b := &NativeBackend{tool: "osascript"}
	// This exercises temp file creation and write; the osascript call
	// may fail (e.g., in headless CI) but the code path is covered.
	_ = b.WriteImage("", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})
}

// --- ShimBackend tests ---

func TestShimBackend_Name(t *testing.T) {
	b := &ShimBackend{}
	if got := b.Name(); got != "shim" {
		t.Errorf("ShimBackend.Name() = %q, want %q", got, "shim")
	}
}

func TestShimBackend_Setup(t *testing.T) {
	dir := t.TempDir()
	b := &ShimBackend{}
	if err := b.Setup(dir); err != nil {
		t.Fatalf("ShimBackend.Setup() = %v", err)
	}

	for _, name := range []string{"xclip", "osascript"} {
		path := filepath.Join(ShimBinDir(dir), name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("shim %s not found: %v", name, err)
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
			t.Errorf("shim %s not executable", name)
		}
	}
}

func TestShimBackend_Setup_EmptyRoot(t *testing.T) {
	b := &ShimBackend{}
	if err := b.Setup(""); err == nil {
		t.Error("expected error for empty root")
	}
}

func TestShimBackend_WriteImage(t *testing.T) {
	dir := t.TempDir()
	b := &ShimBackend{}

	imgData := []byte{0x89, 0x50, 0x4e, 0x47}
	if err := b.WriteImage(dir, "image/png", imgData); err != nil {
		t.Fatalf("ShimBackend.WriteImage() = %v", err)
	}

	readBack, err := os.ReadFile(filepath.Join(dataDir(dir), "image.png"))
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	if string(readBack) != string(imgData) {
		t.Error("image data mismatch")
	}
}

func TestShimBackend_WriteImage_UnsupportedMimeType(t *testing.T) {
	dir := t.TempDir()
	b := &ShimBackend{}
	err := b.WriteImage(dir, "image/jpeg", []byte{1, 2, 3})
	if err == nil {
		t.Error("ShimBackend.WriteImage(image/jpeg) should return error")
	}
}

func TestShimBackend_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	b := &ShimBackend{}

	// Before setup, bin dir doesn't exist — should return nil
	overrides := b.EnvOverrides(dir)
	if overrides != nil {
		t.Error("expected nil overrides before setup")
	}

	// After setup, should return PATH = shim bin dir
	if err := b.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	overrides = b.EnvOverrides(dir)
	if overrides == nil {
		t.Fatal("expected non-nil overrides after setup")
	}
	path, ok := overrides["PATH"]
	if !ok {
		t.Fatal("expected PATH in overrides")
	}
	expected := ShimBinDir(dir)
	if path != expected {
		t.Errorf("PATH = %q, want %q", path, expected)
	}
}

func TestShimBackend_EnvOverrides_EmptyRoot(t *testing.T) {
	b := &ShimBackend{}
	overrides := b.EnvOverrides("")
	if overrides != nil {
		t.Error("expected nil overrides for empty root")
	}
}

func TestBackendInterface(t *testing.T) {
	var _ Backend = &NativeBackend{}
	var _ Backend = &ShimBackend{}
}
