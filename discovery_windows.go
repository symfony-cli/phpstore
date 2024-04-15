/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package phpstore

import (
	"os"
	"path/filepath"
	"regexp"
)

// see https://github.com/composer/windows-setup/blob/master/src/composer.iss
func (s *PHPStore) doDiscover() {
	systemDir := systemDir()
	userHomeDir := userHomeDir()

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

	// Herd
	if userHomeDir != "" {
		s.discoverFromDir(filepath.Join(userHomeDir, ".config", "herd", "bin"), nil, regexp.MustCompile("^php\\d{2}$"), "Herd")
	}
}

func systemDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "C:\\"
	}
	return filepath.VolumeName(cwd) + "\\"
}

func userHomeDir() string {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return userHomeDir
}
