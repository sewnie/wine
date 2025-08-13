// The wine package helps manage a Wineprefix and run Wine.
package wine

import "os/exec"

// Wine returns a Cmd for usage of calling WINE.
func (p *Prefix) Wine(exe string, arg ...string) *Cmd {
	wow := p.bin("wine")
	arg = append([]string{exe}, arg...)
	if wine, err := exec.LookPath(p.bin("wine64")); err == nil {
		wow = wine // prefer wine64 only if possible
	}
	return p.Command(wow, arg...)
}

// Version returns the Wineprefix's Wine version.
func (p *Prefix) Version() string {
	cmd := p.Wine("--version")
	cmd.Stdout = nil // required for Output()
	cmd.Stderr = nil

	ver, err := cmd.Output()
	if len(ver) < 1 || err != nil {
		return "unknown"
	}

	// remove newline
	return string(ver[:len(ver)-1])
}
