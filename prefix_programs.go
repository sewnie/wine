package wine

import (
	"fmt"
	"log/slog"
	"os/exec"
)

const (
	ServerDebug      = "--debug"
	ServerForeground = "--foreground"
	ServerKill       = "--kill"
	ServerPersistent = "--persistent"
	ServerWait       = "--wait"
)

const (
	BootEndSession = "--end-session"
	BootForceExit  = "--force"
	BootInit       = "--init"
	BootKill       = "--kill"
	BootRestart    = "--restart"
	BootShutdown   = "--shutdown"
	BootUpdate     = "--update"
)

// Server runs wineserver with the given arguments.
func (p *Prefix) Server(args ...string) error {
	err := p.Command(p.bin("wineserver"), args...).Run()
	if err == nil {
		return nil
	}
	// 1: server already killed (ServerKill)
	// 2: server already started
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() < 3 {
		return nil
	}
	return err
}

// Boot returns a [Cmd] for wineboot.
func (p *Prefix) Boot(args ...string) *Cmd {
	return p.Wine("wineboot", args...)
}

// Start ensures the Wineprefix's server is running and is
// prepared to run any Wine application. The persistence is
// automatically set to 32.
//
// If the Wineprefix is out of date, it will be updated here.
//
// This procedure is done automatically as necessary by invoking any
// Wine application.
func (p *Prefix) Start() error {
	u, err := p.NeedsUpdate()
	if err != nil {
		slog.Warn("wine: Could not determine Wineprefix update state", "err", err)
	} else if u {
		slog.Info("wine: Updating Wineprefix")
		if err := p.Update(); err != nil {
			return fmt.Errorf("update: %w", err)
		}
	}

	err = p.Server(ServerPersistent, "32")
	if err != nil {
		return err
	}
	return p.Boot(BootRestart).Run()
}

// Kill kills the Wineprefix.
func (p *Prefix) Kill() error {
	return p.Server(ServerKill)
}

// Init returns a [Cmd] for initializating the Wineprefix.
//
// This procedure is done automatically as necessary by invoking any
// Wine application or using [Prefix.Start].
func (p *Prefix) Init() error {
	c := p.Boot(BootInit)
	c.headless = true
	return c.Run()
}

// Update fully re-initalizes the Wineprefix data using Wineboot.
//
// This procedure is done automatically as necessary by invoking any
// Wine application or using [Prefix.Start].
func (p *Prefix) Update() error {
	c := p.Boot(BootUpdate)
	c.headless = true
	return c.Run()
}
