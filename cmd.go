package wine

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
)

var ErrPrefixNotAbs = errors.New("prefix directory is not an absolute path")

// Cmd is is a struct wrapper that overrides methods to better interact
// with a Wineprefix.
//
// For further information, refer to [exec.Cmd].
type Cmd struct {
	*exec.Cmd

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
		return errors.New("already started")
	}

	if c.Err != nil {
		return c.Err
	}

	// Always ensure its created, wine will complain if the root
	// directory doesnt exist
	if c.prefix.dir != "" {
		c.Err = os.MkdirAll(c.prefix.dir, 0o755)
	}

	// Go exec.Command does the same thing done here, but
	// this works nicer with wineserver for an unknown reason,
	// otherwise, wineserver will not fork.
	if c.Stdout != nil && c.Stdout != os.Stdout {
		c.Stdout = nil
		c.pipe(c.prefix.Stdout, c.StdoutPipe)
	}
	if c.Stderr != nil && c.Stderr != os.Stderr {
		c.Stderr = nil
		c.pipe(c.prefix.Stderr, c.StderrPipe)
	}

	return c.Cmd.Start()
}

func (c *Cmd) pipe(dst io.Writer, srcFn func() (io.ReadCloser, error)) {
	if c.Err != nil {
		return
	}
	src, err := srcFn()
	if err != nil {
		c.Err = err
		return
	}

	go func() {
		_, err := io.Copy(dst, src)
		if err != nil && !errors.Is(err, fs.ErrClosed) {
			panic(err)
		}
	}()
}

// Refer to [exec.Cmd.Wait].
func (c *Cmd) Wait() error {
	err := c.Cmd.Wait()

	// Restart the Wineprefix since the new instance will become Headless
	// as a result of wineboot being Headless.
	if len(c.Args) > 1 && c.Args[1] == "wineboot" &&
		slices.Contains(c.Env, "DISPLAY=") {
		_ = c.prefix.Kill()
		_ = c.prefix.Start()
	}

	return err
}
