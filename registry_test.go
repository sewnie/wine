package wine

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRootRegistryKeyExport(t *testing.T) {
	dir := t.TempDir()
	pfx := New(dir, "")

	for _, data := range []struct {
		key  string
		name string
		cont string
		raw  RootRegistryKey
	}{
		{"HKLM", "system.reg", registrySystemData, RootRegistryKey(&RegistryKey{
			Name: "HKEY_LOCAL_MACHINE",
			Subkeys: []*RegistryKey{{Name: "Software", Subkeys: []*RegistryKey{{
				Name:     "Foobar",
				Values:   []RegistryValue{{Name: "Foo", Data: "Bar"}},
				modified: Filetime(0x1dc3e01c855469c),
			}}}},
		})},
		{"HKCU", "user.reg", registryUserData, RootRegistryKey(&RegistryKey{
			Name: "HKEY_CURRENT_USER",
			Subkeys: []*RegistryKey{{Name: "Software", Subkeys: []*RegistryKey{{
				Name:     "Foobar",
				Values:   []RegistryValue{{Name: "Foo", Data: "Bar"}},
				modified: Filetime(0x1dc3e01c855469c),
			}}}},
		})},
	} {
		t.Run(data.key, func(t *testing.T) {
			if err := os.WriteFile(
				filepath.Join(pfx.dir, data.name), []byte(data.cont), 0o644); err != nil {
				t.Errorf("unexpected write error: %v", err)
			}

			key, err := pfx.RegistryOpenKey(data.key)
			if err != nil {
				t.Errorf("unexpected read error: %v", err)
			}

			t.Run("match", func(t *testing.T) {
				if !key.Equal(data.raw) {
					t.Fatalf("expected parsed key match, got %v", registryKeyJSON(key))
				}
			})

			t.Run("export", func(t *testing.T) {
				buf := bytes.Buffer{}

				_ = key.exportSystem(&buf)
				if b := buf.Bytes(); !bytes.Equal(b, []byte(data.cont)) {
					t.Log(string(b))
					t.Fatalf("expected key export match")
				}
			})
		})
	}
}

func TestPrefixRootRegistryKeyExport(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	for _, data := range []struct {
		key  string
		name string
	}{
		{"HKLM", "system.reg"},
		{"HKCU", "user.reg"},
	} {
		t.Run(data.key, func(t *testing.T) {
			org, err := os.ReadFile(filepath.Join(testPfx.dir, data.name))
			if err != nil {
				t.Errorf("unexpected system read error: %v", err)
			}

			buf := bytes.Buffer{}

			key, _ := testPfx.RegistryOpenKey(data.key)
			_ = key.exportSystem(&buf)
			if b := buf.Bytes(); !bytes.Equal(b, []byte(org)) {
				t.Log(string(b))
				t.Fail()
			}
		})
	}
}

const registrySystemData = `WINE REGISTRY Version 2
;; All keys relative to REGISTRY\\Machine

#arch=win64

[Software\\Foobar] 1760553029
#time=1dc3e01c855469c
"Foo"="Bar"
`

const registryUserData = `WINE REGISTRY Version 2
;; All keys relative to REGISTRY\\User\\S-1-5-21-0-0-0-1000

#arch=win64

[Software\\Foobar] 1760553029
#time=1dc3e01c855469c
"Foo"="Bar"
`
