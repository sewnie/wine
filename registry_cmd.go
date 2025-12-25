package wine

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// RegistryAdd adds a new registry key to the Wineprefix with the named key, value,
// type, and data. The value parameter can be empty, to modify the (Default) value.
//
// See [RegistryData] for more details about the type of data.
func (p *Prefix) RegistryAdd(key string, value string, data RegistryData) error {
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

	if _, err := p.registryCmd(args...); err != nil {
		return err
	}
	return nil
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

	if _, err := p.registryCmd(args...); err != nil {
		return err
	}
	return nil
}

// RegistryImport imports keys, values and data from a given registry file
// data into the Wineprefix's registry.
func (p *Prefix) RegistryImport(data string) error {
	// 'reg' does not support reading from stdin, but regedit
	// does, and on an error, a dialog will appear instead.
	cmd := p.Wine("regedit", "/C", "-")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = strings.NewReader(data)
	return cmd.Run()
}

// RegistryKeyImport imports the given key to the Wineprefix. If the root
// key is not a toplevel registry key, an error will be shown to the user
// as a GUI.
func (p *Prefix) RegistryImportKey(key *RegistryKey) error {
	cmd := p.Wine("regedit", "/C", "-")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := key.Export(stdin); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	_ = stdin.Close()

	return cmd.Wait()
}

// RegistryQuery finds the registry key located at path. If the named registry key
// is not found, nil will be returned.
func (p *Prefix) RegistryQuery(path string) (*RegistryKey, error) {
	args := []string{"query", path, "/s"}

	data, err := p.registryCmd(args...)
	if err != nil {
		if strings.Contains(err.Error(), "Unable to find the specified registry key") {
			return nil, nil
		}
		return nil, err
	}

	var reg Registry
	var subkey *RegistryKey

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		out := strings.Split(line, "    ")

		switch len(out) {
		case 1:
			path := out[0]
			if path == "" {
				subkey = nil
				continue
			}
			subkey = reg.queryPath(path, true)
		case 4:
			if subkey == nil {
				return nil, errors.New("wine: value without key")
			}
			data, err := parseCmdData(out[2], out[3])
			if err != nil {
				return nil, fmt.Errorf("subkey %s: %w", out[1], err)
			}
			subkey.SetValue(out[1], data)
		}
	}

	if k := reg.Query(path); k != nil {
		return k, nil
	}
	return nil, errors.New("wine: expected successful key creation")
}

func parseCmdData(dataType string, data string) (RegistryData, error) {
	switch dataType {
	case "REG_SZ", "REG_MULTI_SZ":
		return data, nil
	case "REG_DWORD":
		u, err := strconv.ParseUint(data, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint32(u), nil
	case "REG_QWORD":
		return strconv.ParseUint(data, 0, 64)
	case "REG_BINARY":
		return hex.DecodeString(data)
	case "REG_NONE":
		return byte(0), nil
	}
	return nil, fmt.Errorf("unhandled type %s", dataType)
}

func (p *Prefix) SetDPI(dpi uint) error {
	return p.RegistryAdd(`HKEY_CURRENT_USER\Control Panel\Desktop`, "LogPixels", dpi)
}

func formatRegistryData(data any) (string, string) {
	switch d := data.(type) {
	case string:
		return "REG_SZ", d
	case []string:
		return "REG_MULTI_SZ", strings.Join([]string(d), "\x00") + "\x00\x00"
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

func (p *Prefix) registryCmd(args ...string) ([]byte, error) {
	cmd := p.Wine("reg", args...)
	cmd.Stdout = nil
	b, err := cmd.Output()
	if err != nil {
		// wine reg(1) outputs error to stdout
		if bytes.HasPrefix(b, []byte("reg: ")) {
			return nil, fmt.Errorf("registry error: %s", string(b[5:len(b)-1]))
		}
		return nil, err
	}

	return b, nil
}
