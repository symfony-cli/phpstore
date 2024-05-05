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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// PHPStore stores information about all locally installed PHP versions
type PHPStore struct {
	configDir        string
	versions         versions
	pathVersion      *Version
	seen             map[string]int
	discoveryLogFunc func(msg string, a ...interface{})
}

// New creates a new PHP store
func New(configDir string, reload bool, logger func(msg string, a ...interface{})) *PHPStore {
	s := &PHPStore{
		configDir:        configDir,
		seen:             make(map[string]int),
		discoveryLogFunc: logger,
	}
	if reload {
		os.Remove(filepath.Join(configDir, "php_versions.json"))
	}
	s.loadVersions()
	return s
}

// Versions returns all available PHP versions
func (s *PHPStore) Versions() []*Version {
	return s.versions
}

func (s *PHPStore) IsVersionAvailable(version string) bool {
	// start from the end as versions are always sorted
	for i := len(s.versions) - 1; i >= 0; i-- {
		v := s.versions[i]
		if v.Version == version || strings.HasPrefix(v.Version, version) {
			return true
		}
	}
	return false
}

// BestVersionForDir returns the configured PHP version for the given PHP script
func (s *PHPStore) BestVersionForDir(dir string) (*Version, string, string, error) {
	// forced version?
	if os.Getenv("FORCED_PHP_VERSION") != "" {
		minorPHPVersion := strings.Join(strings.Split(os.Getenv("FORCED_PHP_VERSION"), ".")[0:2], ".")
		if _, err := version.NewVersion(minorPHPVersion); err == nil {
			return s.bestVersion(minorPHPVersion, "internal forced version")
		}
	}

	// .php-version for the currently executed PHP script and up
	if version, foundDir := s.versionForDir(dir, ".php-version"); version != nil {
		return s.bestVersion(string(version), fmt.Sprintf(".php-version from current dir: %s", filepath.Join(foundDir, ".php-version")))
	}

	// composer.json for the currently executed PHP script and up
	if version, foundDir := s.versionForDir(dir, "composer.json"); version != nil {
		var composerJson struct {
			Config struct {
				Platform struct {
					PHP string `json:"php"`
				} `json:"platform"`
			} `json:"config"`
		}
		if err := json.Unmarshal(version, &composerJson); err == nil && composerJson.Config.Platform.PHP != "" {
			return s.bestVersion(composerJson.Config.Platform.PHP, fmt.Sprintf("composer.json from current dir: %s", filepath.Join(foundDir, "composer.json")))
		}
	}

	// .php-version for the current working directory and up
	wd, err := os.Getwd()
	if err == nil {
		if version, foundDir := s.versionForDir(wd, ".php-version"); version != nil {
			return s.bestVersion(string(version), fmt.Sprintf(".php-version from working dir: %s", filepath.Join(foundDir, ".php-version")))
		}
	}

	// .symfony.cloud.yaml for the directory of the script and up
	if version, foundDir := s.versionForDir(dir, ".symfony.cloud.yaml"); version != nil {
		var symfonycloud struct {
			Type string `yaml:"type"`
		}
		if err := yaml.Unmarshal(version, &symfonycloud); err == nil {
			if strings.HasPrefix(symfonycloud.Type, "php:") {
				return s.bestVersion(symfonycloud.Type[4:], fmt.Sprintf("SymfonyCloud: %s", filepath.Join(foundDir, ".symfony.cloud.yaml")))
			}
		}
	}

	// .platform.app.yaml for the directory of the script and up
	if version, foundDir := s.versionForDir(dir, ".platform.app.yaml"); version != nil {
		var platform struct {
			Type string `yaml:"type"`
		}
		if err := yaml.Unmarshal(version, &platform); err == nil {
			if strings.HasPrefix(platform.Type, "php:") {
				return s.bestVersion(platform.Type[4:], fmt.Sprintf("Platform.sh: %s", filepath.Join(foundDir, ".platform.app.yaml")))
			}
		}
	}

	return s.fallbackVersion("")
}

// bestVersion returns the latest patch version for the given major (X), minor (X.Y), or patch (X.Y.Z)
// version can be 7 or 7.1 or 7.1.2
// non-symlinked versions have priority
// If the asked version is a patch one (X.Y.Z) and is not available, the lookup
// will fallback to the last path version for the minor version (X.Y).
// There's no fallback to the major version because PHP is known to occasionally
// break BC in minor versions, so we can't safely fall back.
func (s *PHPStore) bestVersion(versionPrefix, source string) (*Version, string, string, error) {
	warning := ""

	isPatchVersion := false
	pos := strings.LastIndexByte(versionPrefix, '.')
	if pos != strings.IndexByte(versionPrefix, '.') {
		if "99" == versionPrefix[pos+1:] {
			versionPrefix = versionPrefix[:pos]
			pos = strings.LastIndexByte(versionPrefix, '.')
		} else {
			isPatchVersion = true
		}
	}

	// Check if versionPrefix is actually a patch version, if so first do an
	// exact match lookup and fallback to a minor version check
	if isPatchVersion {
		// look for an exact match, the order does not matter here
		for _, v := range s.versions {
			if v.Version == versionPrefix {
				return v, source, "", nil
			}
		}

		// exact match not found, fallback to minor version check
		newVersionPrefix := versionPrefix[:pos]
		warning = fmt.Sprintf(`the current dir requires PHP %s (%s), but this version is not available: fallback to %s`, versionPrefix, source, newVersionPrefix)
		versionPrefix = newVersionPrefix
	}

	// start from the end as versions are always sorted
	for i := len(s.versions) - 1; i >= 0; i-- {
		v := s.versions[i]
		if strings.HasPrefix(v.Version, versionPrefix) {
			return v, source, warning, nil
		}
	}

	return s.fallbackVersion(fmt.Sprintf(`the current dir requires PHP %s (%s), but this version is not available`, versionPrefix, source))
}

func (s *PHPStore) fallbackVersion(warning string) (*Version, string, string, error) {
	if s.pathVersion != nil {
		return s.pathVersion, "default version in $PATH", warning, nil
	}
	if len(s.versions) == 0 {
		return nil, "", warning, errors.New("no PHP binaries detected")
	}
	return s.versions[len(s.versions)-1], "most recent PHP version", warning, nil
}

// loadVersions returns all available PHP versions on this machine
func (s *PHPStore) loadVersions() {
	// disk cache?
	cache := filepath.Join(s.configDir, "php_versions.json")
	if _, err := os.Stat(cache); err == nil {
		if contents, err := os.ReadFile(cache); err == nil {
			var vs versions
			if err := json.Unmarshal(contents, &vs); err == nil {
				for _, v := range vs {
					v.FullVersion, err = version.NewVersion(v.Version)
					if err != nil {
						// someone messed up with the cache
						continue
					}
					if v.IsSystem {
						s.pathVersion = v
					}
					s.versions = append(s.versions, v)
				}
				sort.Sort(s.versions)
				return
			}
		}
	}
	s.discover()
	sort.Sort(s.versions)
	if contents, err := json.MarshalIndent(s.versions, "", "    "); err == nil {
		_ = os.WriteFile(cache, contents, 0644)
	}
}

// addVersion ensures that all versions are unique in the store
func (s *PHPStore) addVersion(version *Version) int {
	idx, ok := s.seen[version.PHPPath]
	sl, _ := filepath.EvalSymlinks(version.PHPPath)
	// double-check to see if that's not just a symlink to another existing version
	if !ok && sl != "" {
		idx, ok = s.seen[sl]
	}

	if !ok {
		s.versions = append(s.versions, version)
		s.seen[version.PHPPath] = len(s.versions) - 1
		if sl != "" {
			s.seen[sl] = len(s.versions) - 1
		}
		return len(s.versions) - 1
	}
	currentScore := 0
	if s.versions[idx].FPMPath != "" {
		currentScore = 2
	} else if s.versions[idx].CGIPath != "" {
		currentScore = 1
	}
	newScore := 0
	if version.FPMPath != "" {
		newScore = 2
	} else if version.CGIPath != "" {
		newScore = 1
	}
	if newScore > currentScore {
		s.versions[idx] = version
	}
	return idx
}

// versionForDir returns the PHP version to use for a given directory
// it tries to go up all directories until it finds a version file
func (s *PHPStore) versionForDir(dir, filename string) ([]byte, string) {
	for {
		if version := s.readVersion(filepath.Join(dir, filename)); version != nil {
			return version, dir
		}
		upDir := filepath.Dir(dir)
		if upDir == dir || upDir == "." {
			break
		}
		dir = upDir
	}
	return nil, ""
}

// readVersion reads the content of a version file (see versionForDir)
func (s *PHPStore) readVersion(file string) []byte {
	if _, err := os.Stat(file); err != nil {
		return nil
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	return bytes.TrimSpace(contents)
}

func (s *PHPStore) log(msg string, a ...interface{}) {
	if s.discoveryLogFunc != nil {
		s.discoveryLogFunc(msg, a...)
	}
}
