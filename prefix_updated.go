package wine

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// NeedsUpdate reports whether the Wineprefix requires a full
// re-initialization, determined by the Wine installation.
//
// Errors that can occur include failure to lookup installation
// and existence of the wineprefix directory.
func (p *Prefix) NeedsUpdate() (bool, error) {
	stamp, err := p.configUpdated()
	if err != nil {
		// Fetching Wineprefix .update-timestamp failed,
		// could be permission denied or another I/O error such
		// as the file being missing.
		// Tell the caller that the Wineprefix is inaccessible,
		// and make Wine handle the error if necssary.
		return true, nil
	}

	updated, err := p.updated()
	if err != nil {
		return true, fmt.Errorf("timestamp: %w", err)
	}
	if updated.IsZero() { // disabled
		return false, nil
	}

	// programs/wineboot/wineboot.c:update_timestamp
	return !stamp.Equal(updated), nil
}

func (p *Prefix) configUpdated() (time.Time, error) {
	w := p.Wine("wine")
	if w.Err != nil {
		return time.Time{}, w.Err
	}

	fi, err := os.Stat(filepath.Join(
		filepath.Dir(w.Path), "../share/wine/wine.inf"))
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

func (p *Prefix) updated() (time.Time, error) {
	b, err := os.ReadFile(filepath.Join(p.dir, ".update-timestamp"))
	if err != nil {
		return time.Time{}, err
	}
	content := strings.TrimSpace(string(b))

	if strings.HasPrefix(content, "disable") {
		return time.Time{}, nil
	}

	v, err := strconv.ParseInt(content, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(v, 0), nil
}
