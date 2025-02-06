package wine

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
)

// Cmd is is a struct wrapper that overrides methods to better interact
// with a Wineprefix.
//
// For further information, refer to [exec.Cmd].
type Cmd struct {
	*exec.Cmd

	prefix string
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

	return &Cmd{
		Cmd:    cmd,
		prefix: p.dir,
	}
}

// Headless removes all window-related variables within the command.
//
// Useful when chaining, and when a command doesn't necessarily need a window.
func (c *Cmd) Headless() *Cmd {
	c.Env = append(c.Environ(),
		"DISPLAY=",
		"WAYLAND_DISPLAY=",
		"WINEDEBUG=fixme-all,-winediag,-systray,-ole,-winediag,-ntoskrnl",
	)
	return c
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
	if c.Process != nil {
		return errors.New("exec: already started")
	}

	if c.Err != nil {
		return c.Err
	}

	// Always ensure its created, wine will complain if the root
	// directory doesnt exist
	if c.prefix != "" {
		c.Err = os.MkdirAll(c.prefix, 0o755)
	}

	// There was a long discussion in #winehq regarding starting wine from
	// Go with os/exec when it's stderr and stdout was set to a file. This
	// behavior causes wineserver to start alongside the process instead of
	// the background, creating issues such as Wineserver waiting for processes
	// alongside the executable - having timeout issues, etc.
	// A stderr pipe will be made to mitigate this behavior when and if
	// the prefix's stderr is non-nil or not os.Stderr.
	if c.Err != nil && c.Stderr != nil && c.Stderr != os.Stderr {
		pfxStderr := c.Stderr
		c.Stderr = nil

		cmdErrPipe, err := c.StderrPipe()
		if err != nil {
			return fmt.Errorf("stderr pipe: %w", err)
		}

		go func() {
			_, err := io.Copy(pfxStderr, cmdErrPipe)
			if err != nil && !errors.Is(err, fs.ErrClosed) {
				panic(err)
			}
		}()
	}

	return c.Cmd.Start()
}
