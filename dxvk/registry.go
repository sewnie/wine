package dxvk

import (
	"slices"

	"github.com/apprehensions/wine"
)

var dllOverridesKey = `HKEY_CURRENT_USER\Software\Wine\DllOverrides`

// Overriden checks if the DXVK DLL overrides have been
// installed in the Wineprefix.
func Overriden(pfx *wine.Prefix) (bool, error) {
	q, err := pfx.RegistryQuery(dllOverridesKey, "")
	if err != nil {
		return false, err
	}

	overrides := []wine.RegistryQuerySubkey{
		{"d3d10core", wine.REG_SZ, "builtin"},
		{"d3d11", wine.REG_SZ, "builtin"},
		{"d3d9", wine.REG_SZ, "builtin"},
		{"dxgi", wine.REG_SZ, "builtin"},
	}

	if len(q) == 0 {
		return false, nil
	}

	allOverrides := true
	for _, o := range overrides {
		if !slices.Contains(q[0].Subkeys, o) {
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
	return pfx.RegistryImport(registryData(`"native,builtin"`))
}

// AddOverrides removes the DXVK DLL overrides to the Wineprefix.
func RemoveOverrides(pfx *wine.Prefix) error {
	return pfx.RegistryImport(registryData(`-`))
}

func registryData(value string) string {
	return "Windows Registry Editor Version 5.00\n\n" +
		"[" + dllOverridesKey + "]\n" +
		`"d3d10core"=` + value + "\n" +
		`"d3d11"=` + value + "\n" +
		`"d3d9"=` + value + "\n" +
		`"dxgi"=` + value
}
