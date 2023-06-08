//go:build !windows
// +build !windows

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
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

func (s *PHPStore) doDiscover() {
	// Defaults
	s.addFromDir("/usr", nil, "*nix")
	s.addFromDir("/usr/local", nil, "*nix")

	homeDir, err := homedir.Dir()
	if err != nil {
		homeDir = ""
		s.log("Could not find home directory: %s", err)
	}

	// phpbrew
	if homeDir != "" {
		s.discoverFromDir(filepath.Join(homeDir, ".phpbrew", "php"), nil, nil, "phpbrew")
	}

	// phpenv
	if homeDir != "" {
		s.discoverFromDir(filepath.Join(homeDir, ".phpenv", "versions"), nil, regexp.MustCompile("^[\\d\\.]+(?:RC|BETA|snapshot)?$"), "phpenv")
	}

	// XAMPP
	s.addFromDir("/opt/lampp", nil, "XAMPP")

	// homebrew
	if out, err := exec.Command("brew", "--cellar").Output(); err == nil {
		prefix := strings.Trim(string(out), "\n")
		// pattern example: php@5.6/5.6.33_9
		s.discoverFromDir(prefix, nil, regexp.MustCompile("^php@(?:[\\d\\.]+)/(?:[\\d\\._]+)$"), "homebrew")
		// pattern example: php/7.2.11
		s.discoverFromDir(prefix, nil, regexp.MustCompile("^php/(?:[\\d\\._]+)$"), "homebrew")
	}

	if runtime.GOOS == "darwin" {
		// Liip PHP https://php-osx.liip.ch/ (pattern example: php5-7.2.0RC1-20170907-205032/bin/php)
		s.discoverFromDir("/usr/local", nil, regexp.MustCompile("^php5\\-[\\d\\.]+(?:RC|BETA)?\\d*\\-\\d+\\-\\d+$"), "Liip PHP")

		// MAMP
		s.discoverFromDir("/Applications/MAMP/bin/php/", nil, regexp.MustCompile("^php[\\d\\.]+(?:RC|BETA)?$"), "MAMP")

		// MacPorts (/opt/local/sbin/php-fpm71, /opt/local/bin/php71)
		s.discoverFromDir("/opt/local", regexp.MustCompile("^php(?:[\\d\\.]+)$"), nil, "MacPorts")
	}

	if runtime.GOOS == "linux" {
		// Ondrej PPA on Linux (bin/php7.2)
		s.discoverFromDir("/usr", regexp.MustCompile("^php(?:[\\d\\.]+)$"), nil, "Ondrej PPA")

		// Remi's RPM repository
		s.discoverFromDir("/opt/remi", nil, regexp.MustCompile("^php(?:\\d+)/root/usr$"), "Remi's RPM")
	}

	// asdf-vm
	var buf bytes.Buffer
	cmd := exec.Command("asdf", "where", "php")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err == nil {
		s.discoverFromDir(filepath.Dir(buf.String()), nil, nil, "asdf-vm")
	}
}
