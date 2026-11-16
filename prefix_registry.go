package wine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// does not seem to change with current user
const sid = `S-1-5-21-0-0-0-1000`

// RootRegistryKey represents a toplevel root registry key at the state
// it was retrieved. It is not equal to the Wineprefix's actual
// registry files unless it was exported.
type RootRegistryKey = *RegistryKey

// RegistryOpenKey instantiates, parses and returns the representation of the
// named registry key. The name must be prefixed with only HKLM or HKCU
// (and their full counterparts).
func (p *Prefix) RegistryOpenKey(name string) (RootRegistryKey, error) {
	var path string
	switch name {
	case "HKEY_LOCAL_MACHINE", "HKLM":
		path = filepath.Join(p.dir, "system.reg")
	case "HKEY_CURRENT_USER", "HKCU":
		path = filepath.Join(p.dir, "user.reg")
	default:
		return nil, nil
	}

	k, err := ParseRegistryFile(path)
	if err != nil {
		return nil, err
	}
	return RootRegistryKey(k), nil
}

// RegistrySaveKey saves the root registry key to its respective registry
// export inside the Wineprefix.
//
// To export a registry to a Wineprefix, the Wineserver must be killed as to
// not conflict with Wineserver's internal registry; it is reccomended to use
// [Prefix.RegistryAdd] or [Prefix.RegistryImportKey] for live usage.
func (p *Prefix) RegistrySaveKey(rk RootRegistryKey) error {
	var path string
	switch rk.Name {
	case "HKEY_LOCAL_MACHINE":
		path = filepath.Join(p.dir, "system.reg")
	case "HKEY_CURRENT_USER":
		path = filepath.Join(p.dir, "user.reg")
	default:
		return os.ErrInvalid
	}

	f, err := os.OpenFile(path,
		os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	return rk.exportSystem(f)
}

// Registry represents the Wineprefix's root registry keys.
type Registry struct {
	CurrentUser RootRegistryKey
	Machine     RootRegistryKey

	pfx *Prefix
}

// Registry parses and returns the registry for the given Wineprefix.
//
// See the commment on [Registry] for more information.
func (p *Prefix) Registry() (*Registry, error) {
	u, err := p.RegistryOpenKey("HKCU")
	if err != nil {
		return nil, fmt.Errorf("user: %w", err)
	}

	m, err := p.RegistryOpenKey("HKLM")
	if err != nil {
		return nil, fmt.Errorf("machine: %w", err)
	}

	return &Registry{
		CurrentUser: u,
		Machine:     m,

		pfx: p,
	}, nil
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
	// nil checks for Registry usage in [Registry.RegistryQuery] command
	case "HKEY_LOCAL_MACHINE", "HKLM":
		if r.Machine == nil {
			r.Machine = RootRegistryKey(&RegistryKey{Name: "HKEY_LOCAL_MACHINE"})
		}
		return r.Machine.queryPath(key, create)
	case "HKEY_CURRENT_USER", "HKCU":
		if r.CurrentUser == nil {
			r.CurrentUser = RootRegistryKey(&RegistryKey{Name: "HKEY_CURRENT_USER"})
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
	if err := r.pfx.RegistrySaveKey(r.CurrentUser); err != nil {
		return fmt.Errorf("user: %w", err)
	}

	if err := r.pfx.RegistrySaveKey(r.Machine); err != nil {
		return fmt.Errorf("machine: %w", err)
	}

	return nil
}
