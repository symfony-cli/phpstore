//go:build !windows

package phpstore

import "path/filepath"

var evalSymlinks = filepath.EvalSymlinks
