// The wine package helps manage a Wineprefix and run Wine.
package wine

// Wine returns a Cmd for usage of calling WINE.
func (p *Prefix) Wine(exe string, arg ...string) *Cmd {
	arg = append([]string{exe}, arg...)
	cmd := p.Command(p.bin("wine"), arg...)
	return cmd
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
