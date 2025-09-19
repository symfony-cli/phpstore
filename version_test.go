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
	"testing"
)

func TestVersion_SupportsFlavor(t *testing.T) {
	testCases := []struct {
		version         *Version
		expectedFlavors []string
	}{
		{
			version: func() *Version {
				v := NewVersion("8.1")
				v.FPMPath = "/usr/bin/php-fpm8.1"
				v.PHPPath = "/usr/bin/php-8.1"
				return v
			}(),
			expectedFlavors: []string{FlavorFPM, FlavorCLI},
		},
		{
			version: func() *Version {
				v := NewVersion("8.2")
				v.CGIPath = "/usr/bin/php-cgi8.1"
				v.PHPPath = "/usr/bin/php-8.1"
				return v
			}(),
			expectedFlavors: []string{FlavorCGI, FlavorCLI},
		},
		{
			version: func() *Version {
				v := NewVersion("8.3")
				v.PHPPath = "/usr/bin/php-8.3"
				return v
			}(),
			expectedFlavors: []string{FlavorCLI},
		},
		{
			version: func() *Version {
				v := NewVersion("8.4")
				v.PHPPath = "/usr/bin/frankenphp"
				v.FrankenPHP = true
				return v
			}(),
			expectedFlavors: []string{FlavorFrankenPHP},
		},
	}
	for _, testCase := range testCases {
		if !testCase.version.SupportsFlavor("") {
			t.Error("version.SupportsFlavor('') should return true, got false")
		}
		for _, flavor := range testCase.expectedFlavors {
			if !testCase.version.SupportsFlavor(flavor) {
				t.Errorf("version.SupportsFlavor(%v) should return true, got false", flavor)
			}
		}
	flavorLoop:
		for _, possibleFlavor := range []string{FlavorCLI, FlavorCGI, FlavorFPM, FlavorFrankenPHP} {
			for _, flavor := range testCase.expectedFlavors {
				if flavor == possibleFlavor {
					continue flavorLoop
				}
			}

			if testCase.version.SupportsFlavor(possibleFlavor) {
				t.Errorf("version.SupportsFlavor(%v) should return false, got true", possibleFlavor)
			}
		}
	}
}
