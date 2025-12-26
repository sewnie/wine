package wine

import (
	"testing"
)

func TestPrefixCmdOperations(t *testing.T) {
	path := `HKCU\Software\Foobar`

	key := NewRegistryKey(path)
	key.Values = []RegistryValue{
		{Name: "Bar", Data: []byte{0xde, 0xad, 0xbe, 0xef}},
		{Name: "Foo", Data: uint32(0xdeadbeef)},
	}

	if err := testPfx.RegistryImportKey(key); err != nil {
		t.Fatalf("unexpected registry import error: %v", err)
	}

	k, err := testPfx.RegistryQuery(path)
	if err != nil {
		t.Fatalf("unexpected registry query error: %v", err)
	}
	if k == nil {
		t.Fatal("expected key query")
	}

	if root := k.Root(); !root.Equal(&RegistryKey{
		Name:    "HKEY_CURRENT_USER",
		Subkeys: []*RegistryKey{{Name: "Software", Subkeys: []*RegistryKey{key}}},
	}) {
		t.Fatalf("expected root key match, got %v", root)
	}

	if !k.Equal(key) {
		t.Fatalf("expected key match %#+v, got %#+v", key, k)
	}

	if err = testPfx.RegistryDelete(path, ""); err != nil {
		t.Fatalf("unexpected registry delete error: %v", err)
	}

	k, err = testPfx.RegistryQuery(path)
	if err != nil {
		t.Fatalf("unexpected registry query error: %v", err)
	}
	if k != nil {
		t.Fatalf("expected key deleted, got %v", k)
	}
}
