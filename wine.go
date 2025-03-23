// The wine package helps manage a Wineprefix and run Wine.
package wine

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	ErrWineNotFound = errors.New("wine64 not found in $PATH or wineroot")
	ErrPrefixNotAbs = errors.New("prefix directory is not an absolute path")
)

// Wine returns a appropiately selected Wine for the Wineprefix.
//
// The Wine executable used is a path to the system or Prefix's Root's 'wine64'
// or 'wine', in preference order, if present.
func (p *Prefix) Wine(exe string, arg ...string) *Cmd {
	wine := p.bin("wine64")
	_, err := os.Stat(wine)
	if err != nil {
		wine = p.bin("wine")
	}

	arg = append([]string{exe}, arg...)
	cmd := p.Command(wine, arg...)

	if (cmd.Err != nil && errors.Is(cmd.Err, exec.ErrNotFound)) ||
		errors.Is(err, os.ErrNotExist) {
		cmd.Err = ErrWineNotFound
	} else if cmd.Err == nil && err != nil {
		cmd.Err = err
	}

	// Wine requires a absolute path for the Wineprefix.
	if p.dir != "" && !filepath.IsAbs(p.dir) {
		cmd.Err = ErrPrefixNotAbs
	}

	return cmd
}

// Version returns the Wineprefix's Wine version.
func (p *Prefix) Version() string {
	cmd := p.Wine("--version")
	cmd.Stdout = nil // required for Output()
	cmd.Stderr = nil

	ver, err := cmd.Output()
	if len(ver) < 0 || err != nil {
		return "unknown"
	}

	// remove newline
	return string(ver[:len(ver)-1])
}
