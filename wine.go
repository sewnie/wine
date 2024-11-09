// The wine package helps manage a Wineprefix and run Wine.
package wine

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	ErrUMURequired = errors.New("umu-run required in $PATH to utilize Proton")
	ErrWineNotFound = errors.New("wine64 not found in $PATH or wineroot")
	ErrPrefixNotAbs = errors.New("prefix directory is not an absolute path")
)

// Wine returns a new Cmd with the prefix's Wine as the named program.
//
// The Wine executable used is a path to the system or Prefix's Root's 'wine64'
// if present. an attempt to resolve for a [UMU Launcher] will be performed.
//
// UMU launcher supports downloading its own UMU-Proton if a proton
// path is not given, but for user preferences, the check will
// only be preferred if a Proton path was set.
//
// [UMU launcher]: https://github.com/Open-Wine-Components/umu-launcher
func (p *Prefix) Wine(exe string, arg ...string) *Cmd {
	wine := "wine64"

	if p.Root != "" {
		wine = filepath.Join(p.Root, "bin", "wine64")

		if _, err := os.Stat(filepath.Join(p.Root, "proton")); err == nil {
			wine = "umu-run"
		} 
	}

	arg = append([]string{exe}, arg...)
	cmd := p.Command(wine, arg...)

	if cmd.Err != nil && errors.Is(cmd.Err, exec.ErrNotFound) {
		if cmd.Args[0] == "umu-run" {
			cmd.Err = ErrUMURequired
		} else {
			cmd.Err = ErrWineNotFound
		}
	}
	
	// Wine requires a absolute path for the wineprefix
	if p.dir != "" && !filepath.IsAbs(p.dir) {
		cmd.Err = ErrPrefixNotAbs
	}

	if cmd.Args[0] == "umu-run" {
		cmd.Env = append(cmd.Environ(),
			"STORE=none",
			"GAMEID=0",
			"PROTONPATH="+p.Root,
		)
	}

	return cmd
}

// Kill kills the Prefix's processes.
func (p *Prefix) Kill() error {
	return p.Wine("wineboot", "-k").Run()
}

// Init preforms initialization for first Wine instance.
func (p *Prefix) Init() error {
	return p.Wine("wineboot", "-i").Run()
}

// Update updates the wineprefix directory.
func (p *Prefix) Update() error {
	return p.Wine("wineboot", "-u").Run()
}

// Version returns the wineprefix's Wine version.
func (p *Prefix) Version() string {
	cmd := p.Wine("--version")
	cmd.Stdout = nil // required for Output()
	cmd.Stderr = nil

	ver, _ := cmd.Output()
	if len(ver) < 0 {
		return "unknown"
	}

	// remove newline
	return string(ver[:len(ver)-1])
}
