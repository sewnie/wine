package wine

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
)

// RegistryType is the type of registry that the wine 'reg' program
// can accept.
type RegistryType string

const (
	REG_SZ        RegistryType = "REG_SZ"
	REG_MULTI_SZ  RegistryType = "REG_MULTI_SZ"
	REG_EXPAND_SZ RegistryType = "REG_EXPAND_SZ"
	REG_DWORD     RegistryType = "REG_DWORD"
	REG_QWORD     RegistryType = "REG_QWORD"
	REG_BINARY    RegistryType = "REG_BINARY"
	REG_NONE      RegistryType = "REG_NONE"
)

// RegistryQueryKey represents a queried registry key.
type RegistryQueryKey struct {
	Key     string
	Subkeys []RegistryQuerySubkey
}

// RegistryQuerySubkey represents a subkey of a [RegistryQueryKey].
type RegistryQuerySubkey struct {
	Name  string
	Type  RegistryType
	Value string
}

// RegistryAdd adds a new registry key to the Wineprefix with the named key, value, type, and data.
func (p *Prefix) RegistryAdd(key, value string, rtype RegistryType, data string) error {
	if key == "" {
		return errors.New("no registry key given")
	}

	return p.Wine("reg", "add", key, "/v", value, "/t", string(rtype), "/d", data, "/f").Run()
}

// RegistryDelete deletes a registry key of the named key and value to be removed
// from the Wineprefix.
func (p *Prefix) RegistryDelete(key, value string) error {
	if key == "" {
		return errors.New("no registry key given")
	}

	return p.Wine("reg", "delete", key, "/v", value, "/f").Run()
}

// RegistryImport imports keys, values and data from a given registry file data into the
// Wineprefix's registry.
func (p *Prefix) RegistryImport(data string) error {
	f, err := os.CreateTemp("", "go_wine_registry_import.reg")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	f.WriteString(data)

	return p.Wine("reg", "import", f.Name()).Run()
}

// RegistryQuery queries and returns all subkeys of the registry key within
// the Wineprefix.
//
// The value parameter can be empty, if wanting to retrieving all of the subkeys
// of the key.
func (p *Prefix) RegistryQuery(key, value string) ([]RegistryQueryKey, error) {
	var q []RegistryQueryKey
	var c *RegistryQueryKey

	args := []string{"query", key, "/s"}
	if value != "" {
		args = append(args, "/v", value)
	}

	cmd := p.Wine("reg", args...)
	cmd.Stdout = nil
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()
		reg := strings.Split(line, "    ")

		switch len(reg) {
		case 1:
			if reg[0] == "" && c != nil {
				q = append(q, *c)
			}
			c = &RegistryQueryKey{Key: reg[0]}
		case 4:
			c.Subkeys = append(c.Subkeys, RegistryQuerySubkey{
				reg[1], RegistryType(reg[2]), reg[3],
			})
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return q, nil
}

func (p *Prefix) SetDPI(dpi int) error {
	return p.RegistryAdd("HKEY_CURRENT_USER\\Control Panel\\Desktop", "LogPixels", REG_DWORD, strconv.Itoa(dpi))
}
