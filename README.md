PHP Store
=========

PHP Store discovers local PHP installations and selects the best installation
for a project. It supports PHP CLI, CGI, FPM, and FrankenPHP installations on
Linux, macOS, and Windows.

Installation
------------

Install the package with Go modules:

```console
go get github.com/symfony-cli/phpstore
```

Basic Usage
-----------

Create a store, then select PHP for a project directory:

```go
package main

import (
    "log"
    "os"
    "path/filepath"

    "github.com/symfony-cli/phpstore"
)

func main() {
    configDir, err := os.UserConfigDir()
    if err != nil {
        log.Fatal(err)
    }

    projectDir, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }

    cacheDir := filepath.Join(configDir, "my-app", "phpstore")
    if err := os.MkdirAll(cacheDir, 0755); err != nil {
        log.Fatal(err)
    }

    store := phpstore.New(cacheDir, false, log.Printf)
    selected, source, warning, err := store.BestVersionForDir(projectDir)
    if err != nil {
        log.Fatal(err)
    }
    if warning != "" {
        log.Printf("warning: %s", warning)
    }

    log.Printf("using PHP %s from %s at %s", selected.Version, source, selected.PHPPath)
}
```

Pass `nil` instead of `log.Printf` to disable discovery logs.

Discovery and Caching
---------------------

`New()` discovers PHP installations in common platform-specific locations,
additional configured directories, and `PATH`. It uses `php-config` when
possible and falls back to running `php --version`.

The constructor arguments are:

- `configDir`: directory used for the `php_versions.json` discovery cache;
- `reload`: when `true`, removes the cache and performs a new discovery;
- `logger`: optional callback that receives formatted discovery messages.

Use `Versions()` to inspect the discovered installations. The result is sorted
by PHP version in ascending order:

```go
for _, version := range store.Versions() {
    log.Printf(
        "PHP %s: CLI=%s FPM=%s CGI=%s",
        version.Version,
        version.PHPPath,
        version.FPMPath,
        version.CGIPath,
    )
}
```

`IsVersionAvailable()` accepts a major, minor, or patch version prefix:

```go
if store.IsVersionAvailable("8.4") {
    log.Print("PHP 8.4 is available")
}
```

Version Selection
-----------------

`BestVersionForDir()` selects PHP using the first matching source in this
order:

1. The `SYMFONY_CLI_PHP_BINARY_PATH` override;
2. A `.php-version` file found from the requested directory upward;
3. `config.platform.php` in a `composer.json` file found from the requested
   directory upward;
4. A `.php-version` file found from the current working directory upward;
5. The PHP type in `.symfony.cloud.yaml` found from the requested directory
   upward;
6. The PHP type in `.platform.app.yaml` found from the requested directory
   upward;
7. The first PHP installation found in `SYMFONY_CLI_PHP_PATH` or `PATH`;
8. The most recent discovered PHP installation.

A version constraint can select a major, minor, or patch release:

```text
8
8.4
8.4.6
```

When an exact patch release is unavailable, selection falls back to the most
recent patch release from the same minor version and returns a warning. It does
not fall back to another minor version.

PHP Flavors
-----------

A version constraint can include a server flavor:

```text
8.4-cli
8.4-cgi
8.4-fpm
8.4-frankenphp
```

The supported flavor constants are `FlavorCLI`, `FlavorCGI`, `FlavorFPM`, and
`FlavorFrankenPHP`. Use `SupportsFlavor()` to inspect support and
`ForceFlavor()` to select a supported flavor explicitly.

Without a flavor constraint, `ServerPath()` and `ServerTypeName()` use
FrankenPHP when applicable, then prefer FPM, CGI, and CLI in that order.

Environment Variables
---------------------

### `SYMFONY_CLI_PHP_PATH`

Adds directories to PHP discovery before the regular `PATH`. Use the operating
system path-list separator to provide several directories. This variable only
expands discovery; it does not select one installation when several match.

```console
export SYMFONY_CLI_PHP_PATH=/opt/php/8.3/bin:/opt/php/8.4/bin
```

On Windows PowerShell:

```powershell
$env:SYMFONY_CLI_PHP_PATH = 'C:\php83;C:\php84'
```

### `SYMFONY_CLI_PHP_BINARY_PATH`

Overrides automatic version selection with one specific PHP CLI binary. The
value must be an absolute path:

```console
export SYMFONY_CLI_PHP_BINARY_PATH=/usr/local/php8.4/bin/php
```

On Windows PowerShell:

```powershell
$env:SYMFONY_CLI_PHP_BINARY_PATH = 'C:\xampp\php\php.exe'
```

The store runs the selected binary with `--version`, resolves symlinks, and
detects companion FPM, CGI, `php-config`, `phpize`, and `phpdbg` binaries when
they follow a conventional installation layout. Standalone PHP binaries are
also supported. The selected installation is included in `Versions()`. A
successful override returns a warning to make the bypass of automatic selection
explicit.

License
-------

PHP Store is available under the GNU Affero General Public License version 3
or later. See [LICENSE](LICENSE) for details.
