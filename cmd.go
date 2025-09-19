package wine

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var ErrPrefixNotAbs = errors.New("prefix directory is not an absolute path")

// Cmd is is a struct wrapper that overrides methods to better interact
// with a Wineprefix.
//
// For further information, refer to [exec.Cmd].
type Cmd struct {
	*exec.Cmd

	// Prevents the command from having a window by removing
	// display environment variables. The wineserver will be
	// ran before the command into foreground, to ensure
	// that the wineserver does not also run headless.
	Headless bool

	prefix *Prefix
}

// Command returns the Cmd struct to execute the named program
// with the given arguments within the Wineprefix.
// It is reccomended to use [Prefix.Wine] to run wine as opposed to Command.
//
// For further information, refer to [exec.Command].
func (p *Prefix) Command(name string, arg ...string) *Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Stderr = p.Stderr
	cmd.Stdout = p.Stdout
	if p.dir != "" {
		cmd.Env = append(cmd.Environ(), "WINEPREFIX="+p.dir)
	}

	// Set cmd.Err even if the path is absolute
	if filepath.Base(name) != name {
		if _, err := exec.LookPath(cmd.Path); err != nil {
			cmd.Err = err
		}
	}

	// Wine requires a absolute path for the Wineprefix.
	if p.dir != "" && !filepath.IsAbs(p.dir) {
		cmd.Err = ErrPrefixNotAbs
	}

	return &Cmd{
		Cmd:    cmd,
		prefix: p,
	}
}

// Quiet sets the command output to nil, used in contexts where errors
// are not to be expected.
func (c *Cmd) Quiet() *Cmd {
	c.Stderr = nil
	c.Stdout = nil
	return c
}

// Refer to [exec.Cmd.Run].
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Refer to [exec.Cmd.Start].
func (c *Cmd) Start() error {
	if c.Headless {
		c.Env = append(c.Environ(),
			"DISPLAY=",
			"WAYLAND_DISPLAY=",
			"WINEDEBUG=fixme-all,-winediag,-systray,-ole,-winediag,-ntoskrnl",
		)

		// Ensure the wineserver is not automatically started with the headless
		// environment variables
		if err := c.prefix.Start(); err != nil {
			return err
		}
	}

	// Always ensure its created, wine will complain if the root
	// directory doesnt exist
	if c.prefix.dir != "" {
		if err := os.MkdirAll(c.prefix.dir, 0o755); err != nil {
			return err
		}
	}

	// https://bugs.winehq.org/show_bug.cgi?id=58707
	if c.Stdout != nil && c.Stdout != os.Stdout {
		c.pipe(&c.Stdout, c.StdoutPipe)
	}
	if c.Stderr != nil && c.Stderr != os.Stderr {
		c.pipe(&c.Stderr, c.StderrPipe)
	}

	return c.Cmd.Start()
}

func (c *Cmd) pipe(pipeDst *io.Writer, pipeFn func() (io.ReadCloser, error)) {
	if c.Err != nil {
		return
	}
	dst := *pipeDst
	*pipeDst = nil
	src, err := pipeFn()
	if err != nil {
		c.Err = err
		return
	}

	go func() {
		_, _ = io.Copy(dst, src)
	}()
}

// Refer to [exec.Cmd.Wait].
func (c *Cmd) Wait() error {
	return c.Cmd.Wait()
}
