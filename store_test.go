package phpstore

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestPHPBinaryPathOverride(t *testing.T) {
	firstRoot := t.TempDir()
	firstPath := filepath.Join(firstRoot, "bin", "php")
	writeTestFile(t, firstPath)
	selectedRoot := t.TempDir()
	selectedPath := filepath.Join(selectedRoot, "bin", "php")
	selectedFPMPath := filepath.Join(selectedRoot, "sbin", "php-fpm")
	writeTestFile(t, selectedPath)
	writeTestFile(t, selectedFPMPath)

	store := newEmpty(t.TempDir(), nil)
	first := NewVersion("8.4.6")
	first.PHPPath = firstPath
	store.addVersion(first)
	selected := NewVersion("8.4.6")
	selected.PHPPath = selectedPath
	store.addVersion(selected)
	var inspectedPath string
	store.phpVersionOutput = func(path string) ([]byte, error) {
		inspectedPath = path
		return []byte("PHP 8.4.6 (cli)"), nil
	}
	t.Setenv(PHPBinaryPathEnvVar, selectedPath)
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".php-version"), []byte("8.3"), 0644); err != nil {
		t.Fatal(err)
	}

	actual, source, warning, err := store.BestVersionForDir(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := resolveTestPath(t, selectedPath)
	if actual.PHPPath != expectedPath {
		t.Fatalf("expected PHP binary %q, got %q", expectedPath, actual.PHPPath)
	}
	expectedFPMPath := resolveTestPath(t, selectedFPMPath)
	if actual.FPMPath != expectedFPMPath {
		t.Fatalf("expected PHP FPM binary %q, got %q", expectedFPMPath, actual.FPMPath)
	}
	if inspectedPath != selectedPath {
		t.Fatalf("expected to inspect %q, got %q", selectedPath, inspectedPath)
	}
	if source != PHPBinaryPathEnvVar+" environment variable" {
		t.Fatalf("unexpected source %q", source)
	}
	if warning != "PHP version selection is overridden by "+PHPBinaryPathEnvVar {
		t.Fatalf("unexpected warning %q", warning)
	}
	if len(store.Versions()) != 2 {
		t.Fatalf("expected two PHP installations, got %d", len(store.Versions()))
	}
}

func TestPHPBinaryPathOverrideResolvesSymlink(t *testing.T) {
	root := t.TempDir()
	realPath := filepath.Join(root, "bin", "php")
	writeTestFile(t, realPath)
	symlinkPath := filepath.Join(t.TempDir(), "php")
	if err := os.Symlink(realPath, symlinkPath); err != nil {
		t.Skipf("symlinks are not available: %s", err)
	}

	store := newEmpty(t.TempDir(), nil)
	store.phpVersionOutput = func(path string) ([]byte, error) {
		return []byte("PHP 8.4.6 (cli)"), nil
	}
	t.Setenv(PHPBinaryPathEnvVar, symlinkPath)

	actual, _, _, err := store.BestVersionForDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := resolveTestPath(t, realPath)
	if actual.PHPPath != expectedPath {
		t.Fatalf("expected resolved PHP binary %q, got %q", expectedPath, actual.PHPPath)
	}
	expectedRoot := resolveTestPath(t, root)
	if actual.Path != expectedRoot {
		t.Fatalf("expected PHP installation root %q, got %q", expectedRoot, actual.Path)
	}
}

func TestPHPBinaryPathOverrideSupportsStandaloneBinary(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "custom-php")
	writeTestFile(t, binaryPath)

	store := newEmpty(t.TempDir(), nil)
	store.phpVersionOutput = func(path string) ([]byte, error) {
		return []byte("PHP 8.5.2 (cli)"), nil
	}
	t.Setenv(PHPBinaryPathEnvVar, binaryPath)

	actual, _, _, err := store.BestVersionForDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := resolveTestPath(t, binaryPath)
	if actual.PHPPath != expectedPath {
		t.Fatalf("expected PHP binary %q, got %q", expectedPath, actual.PHPPath)
	}
	expectedRoot := filepath.Dir(expectedPath)
	if actual.Path != expectedRoot {
		t.Fatalf("expected PHP installation root %q, got %q", expectedRoot, actual.Path)
	}
	if !actual.IsCLIServer() {
		t.Fatalf("expected standalone PHP binary to use the CLI server")
	}
}

func TestPHPBinaryPathOverrideRejectsInvalidPaths(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		store := newEmpty(t.TempDir(), nil)
		store.phpVersionOutput = func(path string) ([]byte, error) {
			t.Fatal("relative PHP binary should not be inspected")
			return nil, nil
		}
		t.Setenv(PHPBinaryPathEnvVar, filepath.Join("bin", "php"))

		_, source, warning, err := store.BestVersionForDir(t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "php binary path must be absolute") {
			t.Fatalf("expected an absolute path error, got %v", err)
		}
		if source != PHPBinaryPathEnvVar+" environment variable" {
			t.Fatalf("unexpected source %q", source)
		}
		if warning != "" {
			t.Fatalf("unexpected warning %q", warning)
		}
	})

	t.Run("not a PHP binary", func(t *testing.T) {
		binaryPath := filepath.Join(t.TempDir(), "custom-php")
		writeTestFile(t, binaryPath)
		store := newEmpty(t.TempDir(), nil)
		store.phpVersionOutput = func(path string) ([]byte, error) {
			return []byte("not PHP"), nil
		}
		t.Setenv(PHPBinaryPathEnvVar, binaryPath)

		_, _, _, err := store.BestVersionForDir(t.TempDir())
		if err == nil || !strings.Contains(err.Error(), "is not a PHP binary") {
			t.Fatalf("expected a PHP binary error, got %v", err)
		}
	})
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0755); err != nil {
		t.Fatal(err)
	}
}

func resolveTestPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := evalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}

	return resolved
}

func TestBestVersion(t *testing.T) {
	store := newEmpty("/dev/null", nil)
	for _, v := range []string{"7.4.33", "8.0.27", "8.1.2", "8.1.14", "8.2.1"} {
		ver := NewVersion(v)
		ver.PHPPath = filepath.Join("/foo", v, "bin", "php")
		store.addVersion(ver)

		if !store.IsVersionAvailable(v) {
			t.Errorf("Version %s should be shown as available", v)
		}
	}

	{
		v := "8.0.26"
		ver := NewVersion(v)
		ver.PHPPath = filepath.Join("/foo", v, "bin", "php")
		ver.FPMPath = filepath.Join("/foo", v, "bin", "php-fpm")
		store.addVersion(ver)

		if !store.IsVersionAvailable(v) {
			t.Errorf("Version %s should be shown as available", v)
		}
	}

	sort.Sort(store.versions)

	{
		bestVersion, _, _, _ := store.bestVersion("8", "testing")
		if bestVersion == nil {
			t.Error("8 requirement should find a best version")
		} else if bestVersion.Version != "8.2.1" {
			t.Errorf("8 requirement should find 8.2.1 as best version, got %s", bestVersion.Version)
		}
	}

	{
		bestVersion, _, _, _ := store.bestVersion("8.1", "testing")
		if bestVersion == nil {
			t.Error("8.1 requirement should find a best version")
		} else if bestVersion.Version != "8.1.14" {
			t.Errorf("8.1 requirement should find 8.1.14 as best version, got %s", bestVersion.Version)
		}
	}

	{
		bestVersion, _, warning, _ := store.bestVersion("8.0.10", "testing")
		if bestVersion == nil {
			t.Error("8.0.10 requirement should find a best version")
		} else if bestVersion.Version != "8.0.27" {
			t.Errorf("8.0.10 requirement should find 8.0.27 as best version, got %s", bestVersion.Version)
		} else if warning == "" {
			t.Error("8.0.10 requirement should trigger a warning")
		}
	}

	{
		bestVersion, _, warning, _ := store.bestVersion("8.0.99", "testing")
		if bestVersion == nil {
			t.Error("8.0.99 requirement should find a best version")
		} else if bestVersion.Version != "8.0.27" {
			t.Errorf("8.0.99 requirement should find 8.0.27 as best version, got %s", bestVersion.Version)
		} else if warning != "" {
			t.Error("8.0.99 requirement should not trigger a warning")
		}
	}

	{
		bestVersion, _, warning, _ := store.bestVersion("8.0-fpm", "testing")
		if bestVersion == nil {
			t.Error("8.0-fpm requirement should find a best version")
		} else if bestVersion.Version != "8.0.26" {
			t.Errorf("8.0-fpm requirement should find 8.0.26 as best version, got %s", bestVersion.Version)
		} else if bestVersion.serverType() != fpmServer {
			t.Error("8.0-fpm requirement should find an FPM expectedFlavors")
		} else if warning != "" {
			t.Error("8.0-fpm requirement should not trigger a warning")
		}
	}
}
