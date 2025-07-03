package phpstore

import (
	"path/filepath"
	"testing"
)

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
}
