package wine

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NeedsUpdate reports whether the Wineprefix requires a full
// re-initialization, determined by the Wine installation.
//
// Errors that can occur include failure to lookup installation
// and existence of the wineprefix directory.
func (p *Prefix) NeedsUpdate() (bool, error) {
	prefixUpdate, err := p.updated()
	if err != nil {
		// Fetching Wineprefix .update-timestamp failed,
		// could be permission denied or another I/O error such
		// as the file being missing.
		// Tell the caller that the Wineprefix is inaccessible,
		// and make Wine handle the error if necssary.
		return true, nil
	}
	if prefixUpdate < 0 { // disabled
		return false, nil
	}

	installStamp, err := p.configUpdated()
	if err != nil {
		return true, fmt.Errorf("config: %w", err)
	}

	// programs/wineboot/wineboot.c:update_timestamp
	return prefixUpdate != installStamp, nil
}

func (p *Prefix) configUpdated() (int64, error) {
	w := p.Wine("wine")
	if w.Err != nil {
		return -1, w.Err
	}

	fi, err := os.Stat(filepath.Join(
		filepath.Dir(w.Path), "../share/wine/wine.inf"))
	if err != nil {
		return -1, err
	}
	return fi.ModTime().Unix(), nil
}

func (p *Prefix) updated() (int64, error) {
	b, err := os.ReadFile(filepath.Join(p.dir, ".update-timestamp"))
	if err != nil {
		return -1, err
	}
	content := strings.TrimSpace(string(b))

	if strings.HasPrefix(content, "disable") {
		return -1, nil
	}

	u, err := strconv.ParseInt(content, 10, 64)
	if err != nil {
		return -1, err
	}

	return u, nil
}
