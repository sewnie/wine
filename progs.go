package wine

import "os/exec"

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

// Server runs wineserver with the given commands.
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

// Start ensures the Wineprefix's server is running.
//
// To have more control over the persistence of the server,
// use p.Server(ServerForeground, ServerPersistent, "2").
func (p *Prefix) Start() error {
	return p.Server()
}

// Kill kills the Wineprefix.
func (p *Prefix) Kill() error {
	return p.Server(ServerKill)
}

// Init returns a [Cmd] for initializating the Wineprefix.
func (p *Prefix) Init() *Cmd {
	c := p.Boot(BootInit)
	c.Headless = true
	return c
}

// Update returns a [Cmd] for updating the Wineprefix.
func (p *Prefix) Update() *Cmd {
	c := p.Boot(BootUpdate)
	c.Headless = true
	return c
}
