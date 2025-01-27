package dxvk

import "os"

// Setenv temporarily tells Wine to use or not to use the DXVK DLLs.
func Setenv(enabled bool) {
	add := ";d3d9,d3d10core,d3d11,dxgi="
	if enabled {
		add += "native"
	} else {
		add += "builtin"
	}
	os.Setenv("WINEDLLOVERRIDES", os.Getenv("WINEDLLOVERRIDES")+add)
}
