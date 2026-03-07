package clipboard

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

func TestSetupShims(t *testing.T) {
	dir := t.TempDir()

	if err := SetupShims(dir); err != nil {
		t.Fatalf("SetupShims: %v", err)
	}

	// Verify shim scripts exist and are executable (Unix only)
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

	// Verify data directory exists
	if _, err := os.Stat(dataDir(dir)); err != nil {
		t.Errorf("data dir not found: %v", err)
	}
}

func TestSetupShims_EmptyRoot(t *testing.T) {
	if err := SetupShims(""); err == nil {
		t.Error("expected error for empty root")
	}
}

func TestSetupShims_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Setup twice should not error
	if err := SetupShims(dir); err != nil {
		t.Fatalf("SetupShims first: %v", err)
	}
	if err := SetupShims(dir); err != nil {
		t.Fatalf("SetupShims second: %v", err)
	}
}

func TestSetupShims_CannotCreateBinDir(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)
	dir := t.TempDir()
	readonlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readonlyDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(readonlyDir, 0755)

	if err := SetupShims(readonlyDir); err == nil {
		t.Error("expected error for read-only directory")
	}
}

func TestSetupShims_CannotCreateDataDir(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)
	dir := t.TempDir()
	// Create bin dir but make parent of data dir read-only after
	shimDir := filepath.Join(dir, shimDirName)
	binDir := filepath.Join(shimDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Make shim dir read-only so data dir creation fails
	os.Chmod(shimDir, 0555)
	defer os.Chmod(shimDir, 0755)

	if err := SetupShims(dir); err == nil {
		t.Error("expected error when data dir cannot be created")
	}
}

func TestSetupShims_CannotWriteXclip(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)
	dir := t.TempDir()
	// Pre-create directories
	binDir := filepath.Join(dir, shimDirName, "bin")
	dataD := filepath.Join(dir, shimDirName, "data")
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(dataD, 0755)
	// Make bin dir read-only so WriteFile fails
	os.Chmod(binDir, 0555)
	defer os.Chmod(binDir, 0755)

	if err := SetupShims(dir); err == nil {
		t.Error("expected error when xclip shim cannot be written")
	}
}

func TestSetupShims_CannotWriteOsascript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("clipboard shims are Unix-only")
	}
	dir := t.TempDir()
	// Pre-create everything and write xclip successfully
	binDir := filepath.Join(dir, shimDirName, "bin")
	dataD := filepath.Join(dir, shimDirName, "data")
	os.MkdirAll(binDir, 0755)
	os.MkdirAll(dataD, 0755)
	// Write xclip (will succeed)
	os.WriteFile(filepath.Join(binDir, "xclip"), []byte("#!/bin/bash"), 0755)
	// Now make bin dir read-only - but xclip already exists
	// We need to prevent osascript write specifically
	// Create osascript as a directory so WriteFile fails
	os.Mkdir(filepath.Join(binDir, "osascript"), 0755)

	if err := SetupShims(dir); err == nil {
		t.Error("expected error when osascript shim cannot be written")
	}
}

func TestSetupShims_ScriptContent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("clipboard shims are Unix-only")
	}
	dir := t.TempDir()
	if err := SetupShims(dir); err != nil {
		t.Fatalf("SetupShims: %v", err)
	}

	// Verify xclip shim contains expected patterns
	xclipContent, err := os.ReadFile(filepath.Join(ShimBinDir(dir), "xclip"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(xclipContent), "#!/bin/bash") {
		t.Error("xclip shim missing shebang")
	}
	if !strings.Contains(string(xclipContent), "image/png") {
		t.Error("xclip shim missing image/png handling")
	}

	// Verify osascript shim contains expected patterns
	osascriptContent, err := os.ReadFile(filepath.Join(ShimBinDir(dir), "osascript"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(osascriptContent), "#!/bin/bash") {
		t.Error("osascript shim missing shebang")
	}
	if !strings.Contains(string(osascriptContent), "PNGf") {
		t.Error("osascript shim missing PNGf handling")
	}
}

func TestWriteImage(t *testing.T) {
	dir := t.TempDir()

	imgData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a} // PNG header

	if err := WriteImage(dir, "image/png", imgData); err != nil {
		t.Fatalf("WriteImage: %v", err)
	}

	// Verify image was written
	imagePath := filepath.Join(dataDir(dir), "image.png")
	readBack, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	if string(readBack) != string(imgData) {
		t.Error("image data mismatch")
	}
}

func TestWriteImage_Overwrite(t *testing.T) {
	dir := t.TempDir()

	// Write first image
	if err := WriteImage(dir, "image/png", []byte{1, 2, 3}); err != nil {
		t.Fatalf("WriteImage first: %v", err)
	}
	// Overwrite with second image
	imgData := []byte{4, 5, 6, 7}
	if err := WriteImage(dir, "image/png", imgData); err != nil {
		t.Fatalf("WriteImage second: %v", err)
	}

	readBack, err := os.ReadFile(filepath.Join(dataDir(dir), "image.png"))
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	if string(readBack) != string(imgData) {
		t.Error("overwrite failed: data mismatch")
	}
}

func TestWriteImage_UnsupportedMimeType(t *testing.T) {
	dir := t.TempDir()
	for _, mime := range []string{"image/jpeg", "image/gif", "text/plain"} {
		if err := WriteImage(dir, mime, []byte{1, 2, 3}); err == nil {
			t.Errorf("WriteImage(%q) should return error for unsupported mime type", mime)
		}
	}
}

func TestWriteImage_CannotCreateDataDir(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)
	dir := t.TempDir()
	readonlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readonlyDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(readonlyDir, 0755)

	if err := WriteImage(readonlyDir, "image/png", []byte{1}); err == nil {
		t.Error("expected error for read-only directory")
	}
}

func TestWriteImage_CannotWriteFile(t *testing.T) {
	dir := t.TempDir()
	// Create data dir
	dDir := filepath.Join(dir, shimDirName, "data")
	os.MkdirAll(dDir, 0755)
	// Create image.png as a directory so WriteFile fails
	os.Mkdir(filepath.Join(dDir, "image.png"), 0755)

	if err := WriteImage(dir, "image/png", []byte{1}); err == nil {
		t.Error("expected error when file cannot be written")
	}
}

func TestWriteImage_EmptyData(t *testing.T) {
	dir := t.TempDir()

	if err := WriteImage(dir, "image/png", []byte{}); err != nil {
		t.Fatalf("WriteImage empty: %v", err)
	}

	readBack, err := os.ReadFile(filepath.Join(dataDir(dir), "image.png"))
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	if len(readBack) != 0 {
		t.Error("expected empty file")
	}
}

func TestShimBinDir(t *testing.T) {
	dir := t.TempDir()
	got := ShimBinDir(dir)
	want := filepath.Join(dir, ".clipboard-shim", "bin")
	if got != want {
		t.Errorf("ShimBinDir: got %q, want %q", got, want)
	}
}

func TestDataDir(t *testing.T) {
	dir := t.TempDir()
	got := dataDir(dir)
	want := filepath.Join(dir, ".clipboard-shim", "data")
	if got != want {
		t.Errorf("dataDir: got %q, want %q", got, want)
	}
}
