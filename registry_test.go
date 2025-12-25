package wine

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry(t *testing.T) {
	dir := t.TempDir()
	pfx := New(dir, "")

	if err := os.WriteFile(filepath.Join(pfx.dir, "system.reg"), []byte(registrySystemData), 0o644); err != nil {
		t.Errorf("unexpected system write error: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pfx.dir, "user.reg"), []byte(registryUserData), 0o644); err != nil {
		t.Errorf("unexpected user write error: %v", err)
	}

	reg, err := pfx.Registry()
	if err != nil {
		t.Errorf("unexpected read error: %v", err)
	}

	exp := &Registry{
		Machine: &RegistryKey{
			Name: "HKEY_LOCAL_MACHINE",
			Subkeys: []*RegistryKey{{Name: "Software", Subkeys: []*RegistryKey{{
				Name:     "Foobar",
				Values:   []RegistryValue{{Name: "Foo", Data: "Bar"}},
				modified: Filetime(0x1dc3e01c855469c),
			}}}},
		},
		CurrentUser: &RegistryKey{
			Name: "HKEY_CURRENT_USER",
			Subkeys: []*RegistryKey{{Name: "Software", Subkeys: []*RegistryKey{{
				Name:     "Foobar",
				Values:   []RegistryValue{{Name: "Foo", Data: "Bar"}},
				modified: Filetime(0x1dc3e01c855469c),
			}}}},
		},
	}

	equalRegistry(t, reg, exp)

	if err := reg.Save(); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	reg, err = pfx.Registry()
	if err != nil {
		t.Errorf("unexpected reread error: %v", err)
	}

	// check if reversible

	t.Run("reversible", func(t *testing.T) {
		equalRegistry(t, reg, exp)
	})
}

func equalRegistry(t *testing.T, reg, exp *Registry) {
	if !reg.CurrentUser.Equal(exp.CurrentUser) {
		t.Fatalf("expected parsed key match, got %v", registryKeyJSON(reg.CurrentUser))
	}

	if !reg.Machine.Equal(exp.Machine) {
		t.Fatalf("expected parsed key match, got %v", registryKeyJSON(reg.Machine))
	}

	buf := bytes.Buffer{}

	_ = reg.Machine.exportSystem(&buf)
	if b := buf.Bytes(); !bytes.Equal(b, []byte(registrySystemData)) {
		t.Log(string(b))
		t.Fatalf("expected machine key export match")
	}

	buf.Reset()
	_ = reg.CurrentUser.exportSystem(&buf)
	if b := buf.Bytes(); !bytes.Equal(b, []byte(registryUserData)) {
		t.Log(string(b))
		t.Fatalf("expected user key export match")
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
