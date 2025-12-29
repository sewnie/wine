package dxvk

import (
	"github.com/sewnie/wine"
	"slices"
)

var overridesRegPath = `HKEY_CURRENT_USER\Software\Wine\DllOverrides`

// Overriden checks if the DXVK DLL overrides have been
// installed in the Wineprefix.
func Overriden(pfx *wine.Prefix) (bool, error) {
	k, err := pfx.RegistryQuery(overridesRegPath)
	if err != nil {
		return false, err
	}

	overrides := []wine.RegistryValue{
		{"d3d10core", "builtin"},
		{"d3d11", "builtin"},
		{"d3d9", "builtin"},
		{"dxgi", "builtin"},
	}

	if len(k.Values) == 0 {
		return false, nil
	}

	allOverrides := true
	for _, o := range overrides {
		if !slices.Contains(k.Values, o) {
			allOverrides = false
		}
	}

	return allOverrides, nil
}

// AddOverrides adds the DXVK DLL overrides to the Wineprefix.
//
// This can be used regardless if DXVK is installed in the
// Wineprefix or not.
func AddOverrides(pfx *wine.Prefix) error {
	return pfx.RegistryImportKey(registryKey("native,builtin"))
}

// AddOverrides removes the DXVK DLL overrides to the Wineprefix.
func RemoveOverrides(pfx *wine.Prefix) error {
	// Delete the overrides
	return pfx.RegistryImportKey(registryKey(nil))
}

func registryKey(dllOverride any) *wine.RegistryKey {
	k := wine.NewRegistryKey(overridesRegPath)
	k.SetValue("d3d10core", dllOverride)
	k.SetValue("d3d11", dllOverride)
	k.SetValue("d3d9", dllOverride)
	k.SetValue("dxgi", dllOverride)
	return k
}
