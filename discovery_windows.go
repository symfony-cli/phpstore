package phpstore

import (
	"os"
	"path/filepath"
	"regexp"
)

// see https://github.com/composer/windows-setup/blob/master/src/composer.iss
func (s *PHPStore) doDiscover() {
	systemDir := systemDir()

	// XAMPP
	s.addFromDir(filepath.Join(systemDir, "xampp", "php"), nil, "XAMPP")

	// Cygwin
	s.addFromDir(filepath.Join(systemDir, "cygwin64", "bin"), nil, "Cygwin")
	s.addFromDir(filepath.Join(systemDir, "cygwin", "bin"), nil, "Cygwin")

	// Chocolatey
	s.discoverFromDir(filepath.Join(systemDir, "tools"), nil, regexp.MustCompile("^php\\d+$"), "Chocolatey")

	// WAMP
	s.discoverFromDir(filepath.Join(systemDir, "wamp64", "bin", "php"), nil, regexp.MustCompile("^php[\\d\\.]+$"), "WAMP")
	s.discoverFromDir(filepath.Join(systemDir, "wamp", "bin", "php"), nil, regexp.MustCompile("^php[\\d\\.]+$"), "WAMP")

	// MAMP
	s.discoverFromDir(filepath.Join(systemDir, "mamp", "bin", "php"), nil, regexp.MustCompile("^php[\\d\\.]+$"), "MAMP")
}

func systemDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "C:\\"
	}
	return filepath.VolumeName(cwd) + "\\"
}
