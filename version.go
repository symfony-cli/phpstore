package phpstore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
)

type serverType int

const (
	fpmServer serverType = iota
	cgiServer
	cliServer
)

// Version stores information about an installed PHP version
type Version struct {
	FullVersion   *version.Version `json:"-"`
	Version       string           `json:"version"`
	Path          string           `json:"path"`
	PHPPath       string           `json:"php_path"`
	FPMPath       string           `json:"fpm_path"`
	CGIPath       string           `json:"cgi_path"`
	PHPConfigPath string           `json:"php_config_path"`
	PHPizePath    string           `json:"phpize_path"`
	PHPdbgPath    string           `json:"phpdbg_path"`
	IsSystem      bool             `json:"is_system"`
}

type versions []*Version

func (vs versions) Len() int           { return len(vs) }
func (vs versions) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs versions) Less(i, j int) bool { return vs[i].FullVersion.LessThan(vs[j].FullVersion) }

func (v *Version) ServerPath() string {
	if v.serverType() == fpmServer {
		return v.FPMPath
	} else if v.serverType() == cgiServer {
		return v.CGIPath
	}
	return v.PHPPath
}

func (v *Version) ServerTypeName() string {
	if v.serverType() == fpmServer {
		return "PHP FPM"
	} else if v.serverType() == cgiServer {
		return "PHP CGI"
	}
	return "PHP CLI"
}

func (v *Version) IsFPMServer() bool {
	return v.serverType() == fpmServer
}

func (v *Version) IsCGIServer() bool {
	return v.serverType() == cgiServer
}

func (v *Version) IsCLIServer() bool {
	return v.serverType() == cliServer
}

func (v *Version) serverType() serverType {
	if v.FPMPath != "" {
		return fpmServer
	} else if v.CGIPath != "" {
		return cgiServer
	}
	return cliServer
}

func (v *Version) setServer(fpm, cgi, phpconfig, phpize, phpdbg string) string {
	msg := fmt.Sprintf("  Found PHP: %s", v.PHPPath)
	fpm = filepath.Clean(fpm)
	if _, err := os.Stat(fpm); err == nil {
		if fpm, err := filepath.EvalSymlinks(fpm); err == nil {
			v.FPMPath = fpm
			msg += fmt.Sprintf(", with FPM: %s", fpm)
		}
	}
	cgi = filepath.Clean(cgi)
	if _, err := os.Stat(cgi); err == nil {
		if cgi, err := filepath.EvalSymlinks(cgi); err == nil {
			v.CGIPath = cgi
			msg += fmt.Sprintf(", with CGI: %s", cgi)
		}
	}
	phpconfig = filepath.Clean(phpconfig)
	if _, err := os.Stat(phpconfig); err == nil {
		if phpconfig, err := filepath.EvalSymlinks(phpconfig); err == nil {
			v.PHPConfigPath = phpconfig
			msg += fmt.Sprintf(", with php-config: %s", phpconfig)
		}
	}
	phpize = filepath.Clean(phpize)
	if _, err := os.Stat(phpize); err == nil {
		if phpize, err := filepath.EvalSymlinks(phpize); err == nil {
			v.PHPizePath = phpize
			msg += fmt.Sprintf(", with phpize: %s", phpize)
		}
	}
	phpdbg = filepath.Clean(phpdbg)
	if _, err := os.Stat(phpdbg); err == nil {
		if phpdbg, err := filepath.EvalSymlinks(phpdbg); err == nil {
			v.PHPdbgPath = phpdbg
			msg += fmt.Sprintf(", with phpdbg: %s", phpdbg)
		}
	}
	return msg
}
