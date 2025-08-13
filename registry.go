package wine

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// RegistryQueryKey represents a queried registry key.
type RegistryQueryKey struct {
	Key     string
	Subkeys []RegistryQuerySubkey
}

// RegistryQuerySubkey represents a subkey of a [RegistryQueryKey].
type RegistryQuerySubkey struct {
	Name string

	// REG_SZ        = string
	// REG_MULTI_SZ  = []string
	// REG_DWORD     = uint32, uint
	// REG_QWORD     = uint64
	// REG_BINARY    = []byte
	// REG_NONE      = byte(0)
	Value any
}

func formatRegistryData(data any) (string, string) {
	switch d := data.(type) {
	case string:
		return "REG_SZ", d
	case []string:
		return "REG_MULTI_SZ", strings.Join([]string(d), "\x00") + "\x00\x00"
	case uint:
		return "REG_DWORD", strconv.FormatUint(uint64(d), 10)
	case uint32:
		return "REG_DWORD", strconv.FormatUint(uint64(d), 10)
	case uint64:
		return "REG_QWORD", strconv.FormatUint(uint64(d), 10)
	case []byte:
		return "REG_BINARY", hex.EncodeToString(d)
	case byte:
		return "REG_NONE", "" // value ignored by reg
	default:
		return "", ""
	}
}

func parseRegistryData(dataType string, data string) (any, error) {
	switch dataType {
	case "REG_SZ", "REG_MULTI_SZ":
		return data, nil
	case "REG_DWORD":
		return strconv.ParseUint(data, 0, 32)
	case "REG_QWORD":
		return strconv.ParseUint(data, 0, 64)
	case "REG_BINARY":
		return hex.DecodeString(data)
	case "REG_NONE":
		return byte(0), nil
	}
	return nil, fmt.Errorf("unhandled type %s", dataType)
}

func (p *Prefix) registry(args ...string) error {
	cmd := p.Wine("reg", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return err
	}

	b, _ := io.ReadAll(out)
	err := cmd.Wait()
	if err == nil {
		return nil
	}

	lines := strings.Split(string(b), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "reg:") {
		return err
	}

	// Remove the "reg:" prefix and the carriage return at the end
	return fmt.Errorf("registry error: %s", lines[0][5:len(lines[0])-1])
}

// RegistryAdd adds a new registry key to the Wineprefix with the named key, value, type, and data.
// The value parameter can be empty, to modify the (Default) value.
//
// See [RegistryQuerySubkey] for more details about the type of data.
func (p *Prefix) RegistryAdd(key string, value string, data any) error {
	if key == "" {
		return errors.New("no registry key given")
	}

	t, d := formatRegistryData(data)
	if t == "" {
		return errors.New("unhandled type var")
	}

	args := []string{"add", key, "/t", t, "/d", d, "/f"}
	if value != "" {
		args = append(args, "/v", value)
	} else {
		args = append(args, "/ve")
	}

	return p.registry(args...)
}

// RegistryDelete deletes a registry key of the named key and value to be removed
// from the Wineprefix. The value parameter can be empty, if wanting to retrieving
// delete the entire key.
func (p *Prefix) RegistryDelete(key, value string) error {
	if key == "" {
		return errors.New("no registry key given")
	}

	args := []string{"delete", key, "/f"}
	if value != "" {
		args = append(args, "/v", value)
	}

	return p.registry(args...)
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

	return p.registry("import", f.Name())
}

// RegistryQuery queries and returns all subkeys of the registry key within
// the Wineprefix. The value parameter can be empty, if wanting to retrieving
// all of the subkeys of the key.
//
// See [RegistryQuerySubkey] for more details about the type of data returned.
func (p *Prefix) RegistryQuery(key, value string) ([]RegistryQueryKey, error) {
	var q []RegistryQueryKey

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

	var c *RegistryQueryKey
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
			value, err := parseRegistryData(reg[2], reg[3])
			if err != nil {
				return nil, fmt.Errorf("subkey %s: %w", reg[1], err)
			}
			c.Subkeys = append(c.Subkeys, RegistryQuerySubkey{
				Name:  reg[1],
				Value: value,
			})
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return q, nil
}

func (p *Prefix) SetDPI(dpi uint) error {
	return p.RegistryAdd(`HKEY_CURRENT_USER\Control Panel\Desktop`, "LogPixels", dpi)
}
