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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

// discover tries to find all PHP versions on the current machine
func (s *PHPStore) discover() {
	s.doDiscover()

	// Under $PATH
	paths := s.pathDirectories(s.configDir)
	s.log("Looking for PHP in the PATH (%s)", paths)
	for _, path := range paths {
		for _, version := range s.findFromDir(path, nil, "PATH") {
			idx := s.addVersion(version)
			// the first one is the default/system PHP binary
			if s.pathVersion == nil {
				s.pathVersion = s.versions[idx]
				s.pathVersion.IsSystem = true
				s.log("  System PHP version (first in PATH)")
			}
		}
	}
}

func (s *PHPStore) discoverFromDir(root string, phpRegexp *regexp.Regexp, pathRegexp *regexp.Regexp, why string) {
	maxDepth := 1
	if pathRegexp != nil {
		maxDepth += strings.Count(pathRegexp.String(), "/")
	}
	filepath.Walk(root, func(path string, finfo os.FileInfo, err error) error {
		if err != nil {
			// prevent panic by handling failure accessing a path
			return nil
		}
		// bypass current directory and non-directory
		if root == path || !finfo.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return errors.WithStack(err)
		}
		// only maxDepth+1 levels of depth
		if strings.Count(rel, string(os.PathSeparator)) > maxDepth {
			return filepath.SkipDir
		}
		s.log("Looking for PHP in %s (%+v) -- %s", path, pathRegexp, why)
		if pathRegexp == nil || pathRegexp.MatchString(rel) {
			s.addFromDir(path, phpRegexp, why)
			return filepath.SkipDir
		}
		return nil
	})
}

func (s *PHPStore) addFromDir(dir string, phpRegexp *regexp.Regexp, why string) {
	for _, v := range s.findFromDir(dir, phpRegexp, why) {
		s.addVersion(v)
	}
}

func (s *PHPStore) findFromDir(dir string, phpRegexp *regexp.Regexp, why string) []*Version {
	s.log("Looking for PHP in %s (%+v) -- %s", dir, phpRegexp, why)

	root := dir
	if filepath.Base(dir) == "bin" {
		dir = filepath.Dir(dir)
	} else if runtime.GOOS != "windows" {
		root = filepath.Join(dir, "bin")
	}

	if phpRegexp == nil {
		if v := s.discoverPHP(dir, "php"); v != nil {
			return []*Version{v}
		}
		return nil
	}

	if _, err := os.Stat(root); err != nil {
		s.log("  Skipping %s as it does not exist", root)
		return nil
	}

	var versions []*Version
	filepath.Walk(root, func(path string, finfo os.FileInfo, err error) error {
		if err != nil {
			// prevent panic by handling failure accessing a path
			return nil
		}
		if root != path && finfo.IsDir() {
			return filepath.SkipDir
		}
		if phpRegexp.MatchString(filepath.Base(path)) {
			if i := s.discoverPHP(dir, filepath.Base(path)); i != nil {
				versions = append(versions, i)
			}
			return nil
		}
		return nil
	})
	return versions
}

func (s *PHPStore) discoverPHP(dir, binName string) *Version {
	// when php-config is not available/useable, fallback to discovering via php, slower but always work
	if runtime.GOOS == "windows" {
		// php-config does not exist on Windows
		return s.discoverPHPViaPHP(dir, binName)
	}

	phpConfigPath := filepath.Join(dir, "bin", strings.Replace(binName, "php", "php-config", 1))
	fi, err := os.Lstat(phpConfigPath)
	if err != nil {
		return s.discoverPHPViaPHP(dir, binName)
	}

	// on Linux, when using alternatives, php-config does not point to right PHP version, so, it cannot be used
	if fi.Mode()&os.ModeSymlink != 0 {
		if path, err := os.Readlink(phpConfigPath); err == nil && strings.Contains(path, "/alternatives/") {
			return s.discoverPHPViaPHP(dir, binName)
		}
	}

	return s.discoverPHPViaPHPConfig(dir, binName)
}

func (s *PHPStore) discoverPHPViaPHP(dir, binName string) *Version {
	php := filepath.Join(dir, "bin", binName)
	if runtime.GOOS == "windows" {
		binName += ".exe"
		php = filepath.Join(dir, binName)
	}

	if _, err := os.Stat(php); err != nil {
		return nil
	}

	var buf bytes.Buffer
	cmd := exec.Command(php, "--version")
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		s.log(`  Unable to run "%s --version: %s"`, php, err)
		return nil
	}
	r := regexp.MustCompile("PHP (\\d+\\.\\d+\\.\\d+)")
	data := r.FindSubmatch(buf.Bytes())
	if data == nil {
		s.log("  %s is not a PHP binary", php)
		return nil
	}
	php = filepath.Clean(php)
	var err error
	php, err = filepath.EvalSymlinks(php)
	if err != nil {
		s.log("  %s is not a valid symlink", php)
		return nil
	}
	v := s.validateVersion(dir, normalizeVersion(string(data[1])))
	if v == nil {
		return nil
	}
	version := &Version{
		Path:        dir,
		Version:     v.String(),
		FullVersion: v,
		PHPPath:     php,
	}

	fpm := filepath.Join(dir, "sbin", strings.Replace(binName, "php", "php-fpm", 1))
	if _, err := os.Stat(fpm); os.IsNotExist(err) {
		fpm = filepath.Join(dir, "bin", strings.Replace(binName, "php", "php-fpm", 1))
	}

	cgi := filepath.Join(dir, "bin", strings.Replace(binName, "php", "php-cgi", 1))
	phpconfig := filepath.Join(dir, "bin", strings.Replace(binName, "php", "php-config", 1))
	phpize := filepath.Join(dir, "bin", strings.Replace(binName, "php", "phpize", 1))
	phpdbg := filepath.Join(dir, "bin", strings.Replace(binName, "php", "phpdbg", 1))
	if runtime.GOOS == "windows" {
		fpm = filepath.Join(dir, strings.Replace(binName, "php", "php-fpm", 1))
		cgi = filepath.Join(dir, strings.Replace(binName, "php", "php-cgi", 1))
		phpconfig = filepath.Join(dir, strings.Replace(binName, "php", "php-config", 1))
		phpize = filepath.Join(dir, strings.Replace(binName, "php", "phpize", 1))
		phpdbg = filepath.Join(dir, strings.Replace(binName, "php", "phpdbg", 1))
	}
	s.log(version.setServer(fpm, cgi, phpconfig, phpize, phpdbg))
	return version
}

func (s *PHPStore) discoverPHPViaPHPConfig(dir, binName string) *Version {
	phpConfig := filepath.Join(dir, "bin", strings.Replace(binName, "php", "php-config", 1))
	file, err := os.Open(phpConfig)
	if err != nil {
		s.log("  Unable to open %s: %s", phpConfig, err)
		return nil
	}
	version := &Version{
		Path: dir,
	}
	sc := bufio.NewScanner(file)
	programPrefix := ""
	programSuffix := ""
	programExtension := ""
	phpCgiBinary := ""
	allFound := 0
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "vernum=") {
			v := s.validateVersion(dir, strings.Trim(sc.Text()[len("vernum="):], `"`))
			if v == nil {
				return nil
			}
			version.Version = v.String()
			version.FullVersion = v
			allFound++
		} else if strings.HasPrefix(sc.Text(), "program_prefix=") {
			programPrefix = strings.Trim(sc.Text()[len("program_prefix="):], `"`)
			allFound++
		} else if strings.HasPrefix(sc.Text(), "program_suffix=") {
			programSuffix = strings.Trim(sc.Text()[len("program_suffix="):], `"`)
			allFound++
		} else if strings.HasPrefix(sc.Text(), "    php_cgi_binary=") {
			phpCgiBinary = strings.Trim(sc.Text()[len("    php_cgi_binary="):], `"`)
			allFound++
		} else if strings.HasPrefix(sc.Text(), "exe_extension=") {
			programExtension = strings.Trim(sc.Text()[len("exe_extension="):], `"`)
			allFound++
		}
	}
	if version.FullVersion == nil {
		s.log("  Unable to find version in %s", phpConfig)
		return nil
	}
	if allFound != 5 {
		s.log("  Unable to parse all information from %s", phpConfig)
		return nil
	}
	if phpCgiBinary == "" {
		phpCgiBinary = fmt.Sprintf("%sphp%s-cgi%s", programPrefix, programSuffix, programExtension)
	} else {
		phpCgiBinary = strings.Replace(phpCgiBinary, "${program_prefix}", programPrefix, 1)
		phpCgiBinary = strings.Replace(phpCgiBinary, "${program_suffix}", programSuffix, 1)
		phpCgiBinary = strings.Replace(phpCgiBinary, "${exe_extension}", programExtension, 1)
		phpCgiBinary = strings.Replace(phpCgiBinary, "${exec_prefix}/", "", 1)
		phpCgiBinary = strings.Replace(phpCgiBinary, "bin/", "", 1)
	}
	version.PHPPath = filepath.Join(version.Path, "bin", fmt.Sprintf("%sphp%s%s", programPrefix, programSuffix, programExtension))
	s.log(version.setServer(
		filepath.Join(version.Path, "sbin", fmt.Sprintf("%sphp-fpm%s%s", programPrefix, programSuffix, programExtension)),
		filepath.Join(version.Path, "bin", phpCgiBinary),
		filepath.Join(version.Path, "bin", fmt.Sprintf("%sphp-config%s%s", programPrefix, programSuffix, programExtension)),
		filepath.Join(version.Path, "bin", fmt.Sprintf("%sphpize%s%s", programPrefix, programSuffix, programExtension)),
		filepath.Join(version.Path, "bin", fmt.Sprintf("%sphpdbg%s%s", programPrefix, programSuffix, programExtension)),
	))
	return version
}

func (s *PHPStore) validateVersion(path, v string) *version.Version {
	if len(v) != 5 {
		s.log("  Unable to parse version %s for PHP at %s: version is non-standard", v, path)
		return nil
	}
	version, err := version.NewVersion(fmt.Sprintf("%c.%s.%s", v[0], v[1:3], v[3:5]))
	if err != nil {
		s.log("  Unable to parse version %s for PHP at %s: %s", v, path, err)
		return nil
	}
	return version
}

func normalizeVersion(v string) string {
	// version is XYYZZ
	parts := strings.Split(v, ".")
	version := parts[0]
	if len(parts[1]) == 1 {
		version += "0"
	}
	version += parts[1]
	if len(parts[2]) == 1 {
		version += "0"
	}
	return version + parts[2]
}

func (s *PHPStore) pathDirectories(configDir string) []string {
	phpShimDir := filepath.Join(configDir, "bin")
	path := os.Getenv("PATH")
	if runtime.GOOS == "windows" {
		path = os.Getenv("Path")
	}
	user := os.Getenv("USERPROFILE")
	dirs := []string{}
	seen := make(map[string]bool)
	for _, dir := range filepath.SplitList(path) {
		dir = strings.Replace(dir, "%%USERPROFILE%%", user, 1)
		edir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			continue
		}
		if edir == phpShimDir {
			continue
		}
		if edir == "" {
			continue
		}
		if _, ok := seen[edir]; ok {
			if dir != edir {
				s.log("  Skipping %s (alias of %s), already in the PATH", dir, edir)
			} else {
				s.log("  Skipping %s, already in the PATH", dir)
			}
			continue
		}
		dirs = append(dirs, edir)
		seen[edir] = true
	}
	return dirs
}
