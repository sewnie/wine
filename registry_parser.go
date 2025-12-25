package wine

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

var (
	unbackslasher = strings.NewReplacer(`\\`, `\`)
	unicoder      = regexp.MustCompile(`\\x([0-9a-fA-F]{4})`)
)

// ParseRegistryFile is a helper for ParseRegistry to parse from a registry file.
func ParseRegistryFile(name string) (*RegistryKey, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var k RegistryKey
	if err := k.Import(f); err != nil {
		return nil, err
	}
	return &k, nil
}

// Import parses the registry file from r and serializes it into a k.
// If parsing from Wine's internal .reg files, the root registry
// will be named, but if parsing from a exported .reg file, the root registry key
// will have no name.
func (k *RegistryKey) Import(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Scan()
	switch header := scanner.Text(); header {
	case headerWine, headerExport:
	default:
		return fmt.Errorf("wine: expected registry header, got %s", header)
	}

	var subkey *RegistryKey
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		switch line[0] {
		case ';':
			if !strings.HasPrefix(line, ";; All keys relative to") {
				continue
			}
			i := strings.LastIndexByte(line, ' ')
			if i <= 0 {
				return strconv.ErrSyntax
			}
			if k.Name != "" {
				return fmt.Errorf("wine: unexpected path directive")
			}

			switch path := line[i+1:]; path {
			case `REGISTRY\\User\\` + sid:
				k.Name = "HKEY_CURRENT_USER"
			case `REGISTRY\\Machine`:
				k.Name = "HKEY_LOCAL_MACHINE"
			default:
				return fmt.Errorf("wine: unknown registry path: %s", path)
			}
		case '#':
			if !strings.HasPrefix(line, "#time=") {
				if line == "#link" {
					subkey.link = true
				}
				continue
			}

			raw := line[strings.IndexByte(line, '=')+1:]
			i, err := strconv.ParseInt(raw, 16, 64)
			if err != nil {
				return err
			}
			subkey.modified = Filetime(i)
		case '[':
			i := strings.IndexByte(line, ']')
			if i <= 0 {
				return strconv.ErrSyntax
			}

			name := `"` + unicoder.ReplaceAllString(line[1:i], `\u$1`) + `"`
			var path string
			err := json.Unmarshal([]byte(name), &path)
			if err != nil {
				return fmt.Errorf("decode path: %w", err)
			}
			subkey = k.Add(path)
			if subkey == nil {
				return errors.New("expected subkey traversal")
			}
		case '"', '@':
			if subkey == nil {
				return errors.New("value without key")
			}
		bytescan:
			if line[len(line)-1] == '\\' {
				line = line[:len(line)-1]
				// read ahead to obtain all multiline bytes, necessary
				// to perform little/big endian serialization
				for scanner.Scan() {
					line += strings.TrimSpace(scanner.Text())
					goto bytescan
				}
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) < 1 {
				return strconv.ErrSyntax
			}
			name, raw := parts[0], parts[1]

			switch name[0] {
			case '@':
				name = ""
			case '"':
				name = name[1 : len(name)-1]
			}

			data, err := parseData(raw)
			if err != nil {
				return fmt.Errorf("parse %s: %w", name, err)
			}

			subkey.Values = append(subkey.Values, RegistryValue{name, data})
		case '\n':
			subkey = nil
		}
	}

	return scanner.Err()
}

func parseData(value string) (RegistryData, error) {
	if len(value) == 0 {
		return nil, errors.New("expected data")
	}
	if value[0] == '"' {
		s, err := strconv.Unquote(value)
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	i := strings.IndexByte(value, ':')
	if i <= 0 {
		return nil, strconv.ErrSyntax
	}

	switch prefix, data := value[:i], value[i+1:]; prefix {
	case "dword":
		v, err := strconv.ParseUint(data, 16, 32)
		if err != nil {
			return nil, fmt.Errorf("dword: %w", err)
		}
		return uint32(v), nil
	case "str(2)":
		return ExpandableString(unquote(data)), nil
	case "str(7)":
		s := strings.Split(unquote(data), `\0`)
		return s[:len(s)-1], nil // foo\0bar\0 -> [foo, bar, ""]
	}

	if !strings.HasPrefix(value[:i], "hex") {
		return nil, fmt.Errorf("unhandled data type: %s", value[:i])
	}

	hex, err := parseBytes(value[i+1:])
	if err != nil {
		return nil, fmt.Errorf("hex: %w", err)
	}
	switch name := value[:i]; name {
	case "hex", "hex(3)":
		return hex, nil
	case "hex(1)":
		return BinaryString(hex), nil
	case "hex(2)":
		s, err := decodeW(hex)
		if err != nil {
			return nil, err
		}
		return ExpandableString(s), nil
	case "hex(4)":
		return DwordLE(binary.LittleEndian.Uint32(hex)), nil
	case "hex(5)":
		return DwordBE(binary.BigEndian.Uint32(hex)), nil
	case "hex(6)":
		s, err := decodeW(hex)
		if err != nil {
			return nil, err
		}
		return Link(s), nil
	case "hex(7)":
		s, err := decodeW(hex)
		if err != nil {
			return nil, err
		}
		v := strings.Split(s, "\x00")
		return v[:len(v)-1], nil // foo\0bar\0 -> [foo, bar, ""]
	case "hex(b)":
		return binary.LittleEndian.Uint64(hex), nil
	default:
		id := strings.IndexByte(name, '(')
		if id <= 0 {
			return nil, fmt.Errorf("unsupported hex type: %s", name)
		}

		v, err := strconv.ParseUint(name[id+1:len(name)-1], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("dword: %w", err)
		}

		return InternalBytes{
			Identifier: uint32(v),
			Data:       hex,
		}, nil
	}
}

func parseBytes(s string) ([]byte, error) {
	byteStrs := strings.Split(s, ",")
	buf := []byte{}

	for _, byteStr := range byteStrs {
		if byteStr == "" {
			continue
		}
		if byteStr[0] == '\\' { // contination line
			break
		}
		b, err := strconv.ParseUint(byteStr, 16, 8)
		if err != nil {
			return nil, err
		}
		buf = append(buf, byte(b))
	}
	return buf, nil
}

// gist.github.com/juergenhoetzel/2d9447cdf5c5b30278adfa7e22ec660e
func decodeW(b []byte) (string, error) {
	ints := make([]uint16, len(b)/2)
	if err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &ints); err != nil {
		return "", err
	}
	if ints[len(ints)-1] == 0 {
		// remove NULL terminator (if present)
		ints = ints[:len(ints)-1]
	}
	return string(utf16.Decode(ints)), nil
}

// Preferrred over strconv.Unquote because it thinks too much
func unquote(s string) string {
	return unbackslasher.Replace(s[1 : len(s)-1])
}
