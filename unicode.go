package wine

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func Unescape(src string) string {
	if src == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(src))

	var high rune = -1
	for i := 0; i < len(src); {
		var cur rune

		if src[i] != '\\' {
			r, width := utf8.DecodeRuneInString(src[i:])
			i += width
			cur = r
			goto surrogate
		}

		i++ // skip '\'
		if i >= len(src) {
			break
		}

		switch src[i] {
		case 'a':
			sb.WriteByte('\a')
		case 'b':
			sb.WriteByte('\b')
		case 'e':
			sb.WriteByte(0x1B)
		case 'f':
			sb.WriteByte('\f')
		case 'n':
			sb.WriteByte('\n')
		case 'r':
			sb.WriteByte('\r')
		case 't':
			sb.WriteByte('\t')
		case 'v':
			sb.WriteByte('\v')
		case '0', '1', '2', '3', '4', '5', '6', '7':
			var val rune
			for k := 0; k < 3 && i < len(src) && (src[i] >= '0' && src[i] <= '7'); k++ {
				val = (val << 3) | rune(src[i]-'0')
				i++
			}
			sb.WriteRune(val)
			continue
		case 'x':
			i++
			if i >= len(src) || !isXDigit(src[i]) {
				sb.WriteByte('x')
				continue
			}
			for k := 0; k < 4 && i < len(src) && isXDigit(src[i]); k++ {
				cur = (cur << 4) | rune(toHex(src[i]))
				i++
			}
			goto surrogate
		case '\\', '"':
			sb.WriteRune(rune(src[i]))
		default:
			// invalid escape
			sb.WriteRune(rune(src[i-1]))
			sb.WriteRune(rune(src[i]))
		}
		i++
		continue // escaped

	surrogate:
		if high != -1 {
			if cur >= 0xdc00 && cur <= 0xdfff {
				sb.WriteRune((high-0xd800)<<10 + (cur - 0xdc00) + 0x10000)
				high = -1
				continue
			}
			sb.WriteRune(utf8.RuneError)
			high = -1
		}

		if cur >= 0xd800 && cur <= 0xdbff {
			high = cur
		} else {
			sb.WriteRune(cur)
		}
	}

	if high != -1 {
		sb.WriteRune(utf8.RuneError)
	}
	return sb.String()
}

func Escape(src string, quote bool, raw bool) string {
	if src == "" {
		return ""
	}

	// decompose surrogates
	u16 := utf16.Encode([]rune(src))
	var sb strings.Builder
	sb.Grow(len(src) * 2)

	for i, n := 0, len(u16); i < n; i++ {
		c := u16[i]

		if raw && c > 127 {
			if !utf16.IsSurrogate(rune(c)) || i+1 >= n {
				sb.WriteRune(rune(c))
				continue
			}

			// compose surrogate for utf16
			r := utf16.DecodeRune(rune(c), rune(u16[i+1]))
			sb.WriteRune(r)
			i++
			continue
		}

		if c > 127 {
			if i+1 < n && u16[i+1] < 128 && isXDigit(byte(u16[i+1])) {
				fmt.Fprintf(&sb, "\\x%04x", c)
			} else {
				fmt.Fprintf(&sb, "\\x%x", c)
			}
			continue
		}

		if c < 32 {
			switch c {
			case '\a':
				sb.WriteString(`\a`)
			case '\b':
				sb.WriteString(`\b`)
			case '\t':
				sb.WriteString(`\t`)
			case '\n':
				sb.WriteString(`\n`)
			case '\v':
				sb.WriteString(`\v`)
			case '\f':
				sb.WriteString(`\f`)
			case '\r':
				sb.WriteString(`\r`)
			case 0x1B:
				sb.WriteString(`\e`)
			default:
				if i+1 < n && u16[i+1] >= '0' && u16[i+1] <= '7' {
					fmt.Fprintf(&sb, "\\%03o", c)
				} else {
					fmt.Fprintf(&sb, "\\%o", c)
				}
			}
			continue
		}

		if !raw && c == '\\' {
			sb.WriteByte('\\')
		}
		if quote && c == '"' {
			sb.WriteByte('\\')
		}
		if !quote && !raw && (c == '[' || c == ']') {
			sb.WriteByte('\\')
		}
		sb.WriteByte(byte(c))
	}

	return sb.String()
}

func isXDigit16(c uint16) bool {
	return c < 128 && isXDigit(byte(c))
}

func isXDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func toHex(c byte) uint16 {
	switch {
	case c >= '0' && c <= '9':
		return uint16(c - '0')
	case c >= 'a' && c <= 'f':
		return uint16(c - 'a' + 10)
	default:
		return uint16(c - 'A' + 10)
	}
}
