package wine

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRegistryOperations(t *testing.T) {
	root := testdata()

	t.Run("addition", func(t *testing.T) {
		k := root.Add("Baz")
		if k == nil {
			t.Fatal("expected key addition")
		}
		value := k.SetValue("Value A", uint32(0xdeadbeef))
		data, ok := value.Data.(uint32)
		if !ok {
			t.Fatalf("expected data type assertion, got %T", value.Data)
		}
		if data != 0xdeadbeef {
			t.Fatalf("expected equality, got %v", data)
		}
	})

	t.Run("query", func(t *testing.T) {
		sub := root.Query("Baz")
		if sub == nil {
			t.Fatal("expected key query")
		}
		if !sub.Equal(&RegistryKey{
			Name:   "Baz",
			Values: []RegistryValue{{"Value A", uint32(0xdeadbeef)}},
		}) {
			t.Fatalf("expected key match, got %v", sub)
		}
	})

	t.Run("edit", func(t *testing.T) {
		k := root.Query("Baz")
		k.SetValue("Value A", uint64(0xdeadbeef))
		if !k.Equal(&RegistryKey{
			Name:   "Baz",
			Values: []RegistryValue{{"Value A", uint64(0xdeadbeef)}},
		}) {
			t.Fatalf("expected key match, got %v", k)
		}
	})

	t.Run("parent", func(t *testing.T) {
		sub := root.Query("Baz")
		if sub == nil {
			t.Fatal("expected child key query")
		}
		parent := sub.Parent()
		if parent == nil {
			t.Fatal("expected parent key")
		}
		knownParent := root.Query("")
		if knownParent == nil {
			t.Fatal("expected root key query")
		}
		if &parent == &knownParent {
			t.Fatalf("expected %p == %p", parent, knownParent)
		}
	})

	t.Run("path", func(t *testing.T) {
		if path := root.Query("Baz").Path(); path != `HKEY_CURRENT_USER\Baz` {
			t.Fatalf("expected absolute key path, got %s", path)
		}
	})

	t.Run("deletion", func(t *testing.T) {
		if !root.Delete("Baz") {
			t.Fatal("expected successful key deletion")
		}
		if k := root.Query("Baz"); k != nil {
			t.Fatal("expected deleted key as nil")
		}
		if data := testdata(); !root.Equal(data) {
			t.Errorf("expected %#+v\ngot %#+v", data, root)
		}
	})
}

func testdata() *RegistryKey {
	root := &RegistryKey{
		Name:     "HKEY_CURRENT_USER",
		modified: Filetime(0x1dc74e5dfeefd32),
		Values: []RegistryValue{
			{"", ""},
			{"Value A", `"C:\Foo" -help`},
		},
	}
	foo := &RegistryKey{
		Name:     "Foo",
		modified: Filetime(0x1dc7347dc3ec40a),
		Values: []RegistryValue{
			{"Value B", []uint8{0xde, 0xad, 0xbe, 0xef, 0x0, 0x0}},
			{"Value C", uint32(0xdeadbeef)},
			{"Value D", []string{"C:\\Foo", "C:\\Bar"}},
			{"Value E", ExpandableString("%APPDATA%\\Foo")},
		},
		parent: root,
	}
	root.Subkeys = append(root.Subkeys, foo)
	bar := &RegistryKey{
		Name:     "Bar",
		modified: Filetime(0x1dc3e01c855469c),
		Values: []RegistryValue{
			{"Value F", uint64(0xdeadbeef)},
			{"Value G", []string{"C:\\Foo", "C:\\Bar"}},
			{"Value H", ExpandableString("%APPDATA%\\Foo")},
			{"Value I", BinaryString{0x48, 0x0, 0x69, 0x0, 0x0, 0x0}},
		},
		parent: foo,
	}
	foo.Subkeys = append(foo.Subkeys, bar)
	baz := &RegistryKey{
		Name:     "Baz",
		modified: Filetime(0x1dc74e26c24986a),
		Values: []RegistryValue{
			{"Value J", DwordLE(0x12345678)},
			{"Value K", DwordBE(0x12345678)},
			{"Value L", []byte{}},
			{"Value M", InternalBytes{0xff, []byte{0xde}}},
		},
		parent: bar,
	}
	bar.Subkeys = append(bar.Subkeys, baz)
	quz := &RegistryKey{
		Name:     "Quz",
		modified: Filetime(0x1dc74efdcaf516c),
		parent:   foo,
	}
	foo.Subkeys = append(foo.Subkeys, quz)
	baz = &RegistryKey{
		Name:     "Baz",
		modified: Filetime(0x1dc74efdcc0807c),
		Values: []RegistryValue{
			{"SymbolicLinkValue", Link(`Foo\Bar\Baz`)},
		},
		link:   true,
		parent: foo,
	}
	foo.Subkeys = append(foo.Subkeys, baz)
	return root
}

func registryKeyJSON(k *RegistryKey) string {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(k)
	return buf.String()
}
