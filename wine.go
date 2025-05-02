// The wine package helps manage a Wineprefix and run Wine.
package wine

import (
	"errors"
	"os/exec"
	"path/filepath"
)

var (
	ErrWineNotFound = errors.New("wine not found in $PATH or wineroot")
	ErrPrefixNotAbs = errors.New("prefix directory is not an absolute path")
)

// Wine returns a Cmd for usage of calling WINE.
func (p *Prefix) Wine(exe string, arg ...string) *Cmd {
	wine, err := exec.LookPath(p.bin("wine"))
	arg = append([]string{exe}, arg...)
	cmd := p.Command(wine, arg...)
	if err != nil {
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
