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
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		// if with ServerKill, already killed
		return nil
	}
	return err
}

// Boot returns a [Cmd] for wineboot.
func (p *Prefix) Boot(args ...string) *Cmd {
	return p.Wine("wineboot", args...)
}

// Kill kills the Wineprefix.
func (p *Prefix) Kill() error {
	return p.Server(ServerKill)
}

// Init returns a [Cmd] for initializating the Wineprefix.
func (p *Prefix) Init() *Cmd {
	return p.Boot(BootInit).Headless()
}

// Update returns a [Cmd] for updating the Wineprefix.
func (p *Prefix) Update() *Cmd {
	return p.Boot(BootUpdate).Headless()
}

// Tricks returns a [Cmd] for winetricks.
func (p *Prefix) Tricks() *Cmd {
	if p.IsProton() {
		// umu-run [winetricks [ARG...]]
		cmd := p.Wine("winetricks")
		if cmd.Args[0] == "umu-run" {
			return cmd
		}
		// fallback to regular winetricks
	}

	cmd := p.Command("winetricks")
	cmd.Env = append(cmd.Environ(),
		"WINE="+p.bin("wine64"),
		"WINESERVER="+p.bin("wineserver"),
	)

	return cmd
}
