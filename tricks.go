package wine

import (
	"strconv"
)

func (p *Prefix) Winetricks() error {
	if p.IsProton() {
		// umu-run [winetricks [ARG...]]
		cmd := p.Wine("winetricks")
		if cmd.Args[0] == "umu-run" {
			return cmd.Run()
		}
		// fallback to regular winetricks
	}

	cmd := p.Command("winetricks")
	cmd.Env = append(cmd.Environ(),
		"WINE="+p.bin("wine64"),
		"WINESERVER="+p.bin("wineserver"),
	)

	return cmd.Run()
}

func (p *Prefix) SetDPI(dpi int) error {
	return p.RegistryAdd("HKEY_CURRENT_USER\\Control Panel\\Desktop", "LogPixels", REG_DWORD, strconv.Itoa(dpi))
}
