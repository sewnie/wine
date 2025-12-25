package wine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// does not seem to change with current user
const sid = `S-1-5-21-0-0-0-1000`

// Registry represents the Wineprefix's root registry keys at the state
// it was retrieved. It is not equal to the Wineprefix's actual
// registry files unless it was exported.
//
// To export a registry to a Wineprefix, the Registry must have
// come from [Registry()], which requires an initialized wineprefix,
// and the Wineserver must be killed as to not conflict with
// Wineserver's internal registry.
//
// To write a RegistryKey to a Wineprefix, you can use either [Prefix.RegistryAdd]
// or [Prefix.RegistryImportKey].
//
// Only HKEY_CURRENT_USER and HKEY_LOCAL_MACHINE are supported
// and queryable.
type Registry struct {
	CurrentUser *RegistryKey
	Machine     *RegistryKey

	pfx *Prefix
}

// Registry parses and returns the registry for the given Wineprefix.
//
// See the commment on [Registry] for more information.
func (p *Prefix) Registry() (*Registry, error) {
	r := Registry{pfx: p}

	k, err := ParseRegistryFile(filepath.Join(p.dir, "system.reg"))
	if err != nil {
		return nil, err
	}
	r.Machine = k

	k, err = ParseRegistryFile(filepath.Join(p.dir, "user.reg"))
	if err != nil {
		return nil, err
	}
	r.CurrentUser = k

	return &r, nil
}

// Query finds the given registry key path in r. nil will be
// returned if no such key was found. The path must be prefixed
// with only HKLM or HKCU (and their full counterparts), as they
// are the only root keys available in Registry.
func (r *Registry) Query(path string) *RegistryKey {
	return r.queryPath(path, false)
}

func (r *Registry) queryPath(path string, create bool) *RegistryKey {
	i := strings.Index(path, `\`)
	if i < 0 {
		i = len(path)
	}

	// List of known registry names and their files (if applicable):
	// - REGISTRY\User\.Default -> userdef.reg
	// - HKEY_LOCAL_MACHINE -> REGISTRY\MACHINE -> system.reg
	// - HKEY_CURRENT_USER -> REGISTRY\User\S-1-5-21-0-0-0-1000 -> user.reg
	// - HKEY_CLASSES_ROOT -> REGISTRY\MACHINE\Software\Classes
	// - HKEY_USERS -> REGISTRY\User
	// - HKEY_CURRENT_CONFIG -> REGISTRY\System\ControlSet001\Enum
	switch root, key := path[:i], path[i+1:]; root {
	case "HKEY_LOCAL_MACHINE", "HKLM":
		if r.Machine == nil {
			r.Machine = &RegistryKey{Name: "HKEY_LOCAL_MACHINE"}
		}
		return r.Machine.queryPath(key, create)
	case "HKEY_CURRENT_USER", "HKCU":
		if r.CurrentUser == nil {
			r.CurrentUser = &RegistryKey{Name: "HKEY_CURRENT_USER"}
		}
		return r.CurrentUser.queryPath(key, create)
	}
	return nil
}

// Save exports and writes r to the Wineprefix's registry files.
// It is assumed that the Registry is serialized from the same
// registry files and must exist.
//
// See the commment on [Registry] for what is exported.
func (r *Registry) Save() error {
	if r.pfx == nil {
		return errors.New("wine: no registry origin")
	}
	s, err := os.OpenFile(filepath.Join(r.pfx.dir, "system.reg"),
		os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open machine: %w", err)
	}
	defer s.Close()

	if err := r.Machine.exportSystem(s); err != nil {
		return fmt.Errorf("export machine: %w", err)
	}

	u, err := os.OpenFile(filepath.Join(r.pfx.dir, "user.reg"),
		os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open user: %w", err)
	}

	if err := r.CurrentUser.exportSystem(u); err != nil {
		return fmt.Errorf("export user: %w", err)
	}

	return nil
}
