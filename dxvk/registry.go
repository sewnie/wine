package dxvk

import (
	"github.com/sewnie/wine"
	"slices"
)

var dllOverridesKey = `HKEY_CURRENT_USER\Software\Wine\DllOverrides`

// Overriden checks if the DXVK DLL overrides have been
// installed in the Wineprefix.
func Overriden(pfx *wine.Prefix) (bool, error) {
	k, err := pfx.RegistryQuery(dllOverridesKey)
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
