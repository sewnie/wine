package wine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
)

const (
	headerWine   = `WINE REGISTRY Version 2`
	headerExport = `Windows Registry Editor Version 5.00`
)

// Export writes the regedit export of k to w. Any error regarding
// formatting a type will not be returned if k's origin was serialized
// from ParseRegistry.
//
// If there are any registry keys with no values and subkeys, it will
// be marked as deleted.
//
// Registry keys that are links to other keys will not be exported here.
func (k *RegistryKey) Export(w io.Writer) error {
	_, err := io.WriteString(w, headerExport+"\n")
	if err != nil {
		return err
	}

	return k.export(false, w)
}

func (k *RegistryKey) exportSystem(w io.Writer) error {
	_, err := io.WriteString(w, headerWine+"\n;; All keys relative to ")
	if err != nil {
		return err
	}
	switch k.Name {
	case "HKEY_CURRENT_USER":
		_, err = io.WriteString(w, `REGISTRY\\User\\`+sid)
	case `HKEY_LOCAL_MACHINE`:
		_, err = io.WriteString(w, `REGISTRY\\Machine`)
	}
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n\n#arch=win64\n"); err != nil {
		return err
	}

	return k.export(true, w)
}

func (k *RegistryKey) export(wine bool, w io.Writer) error {
	// TODO: support links for regedit export
	if k.link && !wine {
		return nil
	}

	if len(k.Values) == 0 && len(k.Subkeys) == 0 && !wine {
		_, err := fmt.Fprintf(w, "\n[-%s]\n", Escape(k.Path(), false, !wine))
		return err
	}
	if len(k.Values) > 0 || (wine && !k.modified.IsZero()) {
		var err error
		if !wine {
			// If exporting, the raw bytes are given out
			_, err = fmt.Fprintf(w, "\n[%s]\n", Escape(k.Path(), false, !wine))
		} else {
			_, err = fmt.Fprintf(w, "\n[%s] %d\n#time=%x\n",
				Escape(k.pathWine(), false, !wine), k.modified.Unix(), k.modified)
		}
		if err != nil {
			return err
		}
	}
	if k.link {
		if _, err := io.WriteString(w, "#link\n"); err != nil {
			return err
		}
	}
	for _, v := range k.Values {
		err := v.export(w, wine)
		if err != nil {
			return err
		}

	}

	for _, sk := range k.Subkeys {
		err := sk.export(wine, w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rv RegistryValue) export(w io.Writer, wine bool) error {
	var payload []byte
	var (
		err error
		pos int
	)

	if wine && rv.Data == nil {
		return nil
	}

	if rv.Name != "" {
		pos, err = fmt.Fprintf(w, `"%s"=`, rv.Name)
	} else {
		pos, err = io.WriteString(w, `@=`)
	}
	if err != nil {
		return err
	}
	// Add now as this will only be used when printing hex(n) or str(n),
	// and any other case is disallowed.
	pos += 6

	switch d := rv.Data.(type) {
	case nil:
		_, err = io.WriteString(w, "-")
	case string:
		// Dumps normal and quotes in server/registry.c
		_, err = io.WriteString(w, `"`+Escape(d, true, false)+`"`)
	case ExpandableString:
		if wine {
			_, err = io.WriteString(w, `str(2):"`+Escape(string(d), true, false)+`"`)
			break
		}
		_, err = io.WriteString(w, `hex(2):`)
		payload = encodeW(string(d) + "\x00")
	case []string:
		if !wine {
			_, err = io.WriteString(w, `hex(7):`)
			payload = encodeW(strings.Join(d, "\x00") + "\x00\x00")
			break
		}
		_, err = io.WriteString(w, `str(7):"`)
		if err != nil {
			return err
		}
		for _, s := range d {
			_, err := io.WriteString(w, Escape(s, false, false)+`\0`)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, `"`)
	case uint32:
		_, err = fmt.Fprintf(w, "dword:%08x", d)
	case uint64:
		_, err = io.WriteString(w, "hex(b):")
		payload = make([]byte, 8)
		binary.LittleEndian.PutUint64(payload, uint64(d))
	case []byte:
		_, err = io.WriteString(w, "hex:")
		payload = d
		pos -= 3 // hex:
	case BinaryString:
		_, err = io.WriteString(w, "hex(1):")
		payload = d
	case DwordLE:
		_, err = io.WriteString(w, "hex(4):")
		payload = make([]byte, 4)
		binary.LittleEndian.PutUint32(payload, uint32(d))
	case DwordBE:
		_, err = io.WriteString(w, "hex(5):")
		payload = make([]byte, 4)
		binary.BigEndian.PutUint32(payload, uint32(d))
	case Link:
		_, err = io.WriteString(w, `hex(6):`)
		payload = encodeW(string(d))
	case InternalBytes:
		_, err = fmt.Fprintf(w, "hex(%08x):", d.Identifier)
		pos += 7 // ffffff, first n already included
		payload = d.Data
	default:
		return fmt.Errorf("wine: unhandled registry value type: %T", d)
	}
	if err != nil {
		return err
	}

	for i, byte := range payload {
		_, err := fmt.Fprintf(w, "%02x", byte)
		pos += 3
		if i < len(payload)-1 && err == nil {
			_, err = io.WriteString(w, ",")
			if pos+1 > 76 && err == nil {
				_, err = io.WriteString(w, "\\\n  ")
				pos = 2
			}
		}
		if err != nil {
			return err
		}
	}

	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	return nil
}

func encodeW(s string) []byte {
	buf := bytes.Buffer{}
	_ = binary.Write(&buf, binary.LittleEndian, utf16.Encode([]rune(s)))
	return buf.Bytes()
}
