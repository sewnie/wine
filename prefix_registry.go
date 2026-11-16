package wine

import (
	"fmt"
	"os"
	"path/filepath"
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

	// List of known registry names and their files (if applicable):
	// - REGISTRY\User\.Default -> userdef.reg
	// - HKEY_LOCAL_MACHINE -> REGISTRY\MACHINE -> system.reg
	// - HKEY_CURRENT_USER -> REGISTRY\User\S-1-5-21-0-0-0-1000 -> user.reg
	// - HKEY_CLASSES_ROOT -> REGISTRY\MACHINE\Software\Classes
	// - HKEY_USERS -> REGISTRY\User
	// - HKEY_CURRENT_CONFIG -> REGISTRY\System\ControlSet001\Enum
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
// export inside the Wineprefix. It is assumed that the Registry is serialized
// from the same registry files and must exist.
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
