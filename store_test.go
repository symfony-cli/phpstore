package phpstore

import (
	"path/filepath"
	"testing"
)

func TestBestVersion(t *testing.T) {
	store := New("/dev/null", false, nil)
	for _, v := range []string{"7.4.33", "8.0.27", "8.1.2", "8.1.14", "8.2.1"} {
		store.addVersion(&Version{
			Version: v,
			PHPPath: filepath.Join("/foo", v, "bin", "php"),
		})

		if !store.IsVersionAvailable(v) {
			t.Errorf("Version %s should be shown as available", v)
		}
	}

	{
		bestVersion, _, _, _ := store.bestVersion("8", "testing")
		if bestVersion == nil {
			t.Error("8 requirement should find a best version")
		} else if bestVersion.Version != "8.2.1" {
			t.Error("8 requirement should find 8.2.1 as best version")
		}
	}

	{
		bestVersion, _, _, _ := store.bestVersion("8.1", "testing")
		if bestVersion == nil {
			t.Error("8.1 requirement should find a best version")
		} else if bestVersion.Version != "8.1.14" {
			t.Error("8.1 requirement should find 8.1.14 as best version")
		}
	}

	{
		bestVersion, _, warning, _ := store.bestVersion("8.0.10", "testing")
		if bestVersion == nil {
			t.Error("8.0.10 requirement should find a best version")
		} else if bestVersion.Version != "8.0.27" {
			t.Error("8.0.10 requirement should find 8.0.27 as best version")
		} else if warning == "" {
			t.Error("8.0.10 requirement should trigger a warning")
		}
	}

	{
		bestVersion, _, warning, _ := store.bestVersion("8.0.99", "testing")
		if bestVersion == nil {
			t.Error("8.0.99 requirement should find a best version")
		} else if bestVersion.Version != "8.0.27" {
			t.Error("8.0.99 requirement should find 8.0.27 as best version")
		} else if warning != "" {
			t.Error("8.0.99 requirement should not trigger a warning")
		}
	}
}
