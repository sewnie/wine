// The wine package helps manage a Wineprefix and run Wine.
package wine

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Wine returns a Cmd for usage of calling WINE.
func (p *Prefix) Wine(exe string, arg ...string) *Cmd {
	wow := p.bin("wine")
	arg = append([]string{exe}, arg...)
	if wine, err := exec.LookPath(p.bin("wine64")); err == nil {
		wow = wine // prefer wine64 only if possible
	}
	return p.Command(wow, arg...)
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

func (p *Prefix) LibDir() string {
	// Wine's build time configured LIBDIR is actually
	// unretrievable without introducing execution overhead.
	// Assume that the libdir is PREFIX/lib{,64}.
	bindir := filepath.Dir(p.Wine("").Path)
	prefix := filepath.Dir(bindir)
	for _, lib := range []string{"lib", "lib64"} {
		rel, err := filepath.Rel(bindir, filepath.Join(prefix, lib))
		if err != nil {
			continue
		}
		return filepath.Join(bindir, rel)
	}
	panic("unreachable")
}

func (p *Prefix) archDir(pe bool) string {
	// Go adaptation of tools/tools.h:get_arch_dir
	cpu, ok := map[string]string{
		"386":   "i386",
		"amd64": "x86_64",
		"arm":   "arm",
		"arm64": "aarch64",
	}[runtime.GOARCH]
	if !ok {
		return ""
	}
	abi := "windows"
	if !pe {
		abi = "unix"
	}
	return fmt.Sprintf("/%s-%s", cpu, abi)
}

// func (p *Prefix) DllPath() string {
// 	libdir := ""
// 	filepath.Join(
// 		filepath.Dir(p.Wine("").Path),
// 		"../../dlls/"
// }
