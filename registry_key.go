package wine

import (
	"reflect"
	"slices"
	"strings"
)

// RegistryKey represents a relative, offline Wine registry key with its
// known values and subkeys.
//
// It is not reccomended to iterate over the Subkeys field to modify it,
// use [RegistryKey.Add] and [RegistryKey.Delete].
//
// Symlinked registry keys are unsupported. They always contain a subkey
// with the name SymbolicLinkValue and an absolute registry path
// such as '\Registry\Machine\Software\Classes\AppsId' encoded in UTF16LE.
type RegistryKey struct {
	Name    string
	Values  []RegistryValue
	Subkeys []*RegistryKey

	parent   *RegistryKey
	modified Filetime
	link     bool
}

// RegistryValue represents a known registry key's value pairs.
//
// The (Default) key will have the name as empty.
type RegistryValue struct {
	Name string
	Data RegistryData
}

// NewRegistryKey creates a registry key and its parents based
// on the absolute registry path.
func NewRegistryKey(path string) *RegistryKey {
	i := strings.Index(path, `\`)
	if i < 0 {
		i = len(path)
	}

	parent := RegistryKey{Name: path[:i]}
	switch parent.Name {
	case "HKLM":
		parent.Name = "HKEY_LOCAL_MACHINE"
	case "HKCU":
		parent.Name = "HKEY_CURRENT_USER"
	}
	return parent.Add(path[i+1:])
}

// GetValue finds the a registry value with the given name in k. If it is
// not found, nil will be returned.
func (k *RegistryKey) GetValue(name string) *RegistryValue {
	for i, v := range k.Values {
		if v.Name == name {
			return &k.Values[i]
		}
	}
	return nil
}

// SetValue sets a named value in k with the specified data. The name may be
// an empty string to specify the (Default) key, and the data may be nil
// to represent no data (REG_NONE).
//
// If the named value already exists in k, only the data will be set, otherwise
// a new value will be added to k with the given name and data.
func (k *RegistryKey) SetValue(name string, data RegistryData) *RegistryValue {
	if v := k.GetValue(name); v != nil {
		v.Data = data
		return v
	}
	k.Values = append(k.Values, RegistryValue{name, data})
	return &k.Values[len(k.Values)-1]
}

// Add will find the registry key located at path, relative to k,
// and creates any parent key if necesary.
func (k *RegistryKey) Add(path string) *RegistryKey {
	return k.queryPath(path, true)
}

// Parent returns k's parent registry key. The parent can be null
// if k is a root registry key such as HKEY_CURRENT_USER.
func (k *RegistryKey) Parent() *RegistryKey {
	return k.parent
}

// Parent returns k's root registry key (greatest parent).
func (k *RegistryKey) Root() (parent *RegistryKey) {
	for cur := k; cur != nil; cur = cur.parent {
		parent = cur
	}
	return
}

// Path returns the path to itself up to the root registry key.
func (k *RegistryKey) Path() string {
	if k.parent == nil {
		return k.Name
	}
	return k.parent.Path() + `\` + k.Name
}

func (k *RegistryKey) pathWine() (path string) {
	if k.parent == nil {
		return ""
	}

	path = k.Name
	// exclude the root
	for k.parent != nil && k.parent.parent != nil {
		k = k.parent
		path = k.Name + `\\` + path
	}
	return
}

// Query finds the given registry key path relative to k. nil will be returned
// if no key was found.
func (k *RegistryKey) Query(path string) *RegistryKey {
	return k.queryPath(path, false)
}

// Delete removes the named registry key path relative to k. If deletion
// was successful and the key was found in k, true will be returned, otherwise
// false will be returned.
func (k *RegistryKey) Delete(path string) bool {
	query := k.Query(path)
	if query == nil {
		return false
	}
	if query.parent == nil {
		k = nil
	}

	for i, subkey := range query.parent.Subkeys {
		if subkey != query {
			continue
		}
		query.parent.Subkeys = append(
			query.parent.Subkeys[:i],
			query.parent.Subkeys[i+1:]...)
		return true
	}
	panic("wine: subkey successfully traversed but is missing in parent")
}

func (k *RegistryKey) queryPath(path string, create bool) *RegistryKey {
	if path == "" {
		return k
	}

	current := k
segment:
	for _, segment := range strings.Split(path, `\`) {
		// Iterate backwards as the most recently added key would
		// be last, useful in parsing.
		for _, subkey := range slices.Backward(current.Subkeys) {
			if subkey.Name == segment {
				current = subkey
				continue segment
			}
		}
		if !create {
			return nil
		}
		current.Subkeys = append(current.Subkeys, &RegistryKey{
			Name:   segment,
			parent: current,
		})
		current = current.Subkeys[len(current.Subkeys)-1]
	}
	return current
}

// This is preferred over [reflect.DeepEqual] as there are private pointer
// properties.
func (k *RegistryKey) Equal(b *RegistryKey) bool {
	if k == nil || b == nil {
		return true
	}
	if k.Name != b.Name || k.modified != b.modified || k.link != b.link {
		return false
	}
	if len(k.Values) != len(b.Values) {
		return false
	}
	for i := range k.Values {
		if !reflect.DeepEqual(k.Values[i], b.Values[i]) {
			return false
		}
	}
	if len(k.Subkeys) != len(b.Subkeys) {
		return false
	}
	for i := range k.Subkeys {
		if !k.Subkeys[i].Equal(b.Subkeys[i]) {
			return false
		}
	}
	return true
}
