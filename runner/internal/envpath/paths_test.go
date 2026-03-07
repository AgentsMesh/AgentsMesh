package envpath

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestPrependToPath_ExactMatch_NoSubstringFalsePositive(t *testing.T) {
	// "/usr/local/bin" is a substring of "/usr/local/bin/extra" but they are
	// different PATH elements. PrependToPath must NOT skip "/usr/local/bin".
	sep := string(os.PathListSeparator)
	current := "/usr/local/bin/extra" + sep + "/usr/bin"

	result := PrependToPath(current, "/usr/local/bin")
	if !strings.HasPrefix(result, "/usr/local/bin"+sep) {
		t.Errorf("expected /usr/local/bin to be prepended, got: %s", result)
	}
}

func TestPrependToPath_SkipsDuplicateExactElement(t *testing.T) {
	sep := string(os.PathListSeparator)
	current := "/usr/local/bin" + sep + "/usr/bin"

	result := PrependToPath(current, "/usr/local/bin")
	if result != current {
		t.Errorf("expected no change when dir already exists exactly, got: %s", result)
	}
}

func TestPrependToPath_EmptyDirsSkipped(t *testing.T) {
	current := "/usr/bin"

	result := PrependToPath(current, "", "")
	if result != current {
		t.Errorf("expected no change for empty dirs, got: %s", result)
	}
}

func TestPrependToPath_MultipleNewDirs(t *testing.T) {
	sep := string(os.PathListSeparator)
	current := "/usr/bin"

	result := PrependToPath(current, "/opt/a", "/opt/b")

	// Dirs are prepended in order: /opt/a should come before /opt/b
	expected := "/opt/a" + sep + "/opt/b" + sep + current
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPrependToPath_MixedNewAndExisting(t *testing.T) {
	sep := string(os.PathListSeparator)
	current := "/usr/bin" + sep + "/opt/existing"

	result := PrependToPath(current, "/opt/new", "/opt/existing")

	// /opt/existing already present → only /opt/new prepended
	expected := "/opt/new" + sep + current
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPrependToPath_EmptyCurrent(t *testing.T) {
	sep := string(os.PathListSeparator)
	result := PrependToPath("", "/usr/bin")

	expected := "/usr/bin" + sep
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPrependToPath_WindowsSemicolonSeparator(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	current := `C:\Windows\System32;C:\Windows`
	result := PrependToPath(current, `C:\Go\bin`)

	if !strings.HasPrefix(result, `C:\Go\bin;`) {
		t.Errorf("expected C:\\Go\\bin to be prepended with semicolon, got: %s", result)
	}

	// Already existing → no change
	result2 := PrependToPath(current, `C:\Windows\System32`)
	if result2 != current {
		t.Errorf("expected no change for existing dir, got: %s", result2)
	}
}
