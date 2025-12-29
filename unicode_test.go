package wine

import (
	"testing"
)

func TestUnescape(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "surrogate",
			input:    `\xd83c\xdf0e\xd83c\xdf0f\xd83c\xdf0d`,
			expected: "🌎🌏🌍",
		},
		{
			name:     "escaped",
			input:    `\t\n\r\2\101\x0041B\x20\z`,
			expected: "\t\n\r\x02A\x41B \\z",
		},
		{
			name:     "backslash",
			input:    `\\foo\\\\bar`,
			expected: `\foo\\bar`,
		},

		{
			name:     "unquote",
			input:    `\"C:\\Foo\" -help`,
			expected: `"C:\Foo" -help`,
		},
		{
			name:     "escaped octal",
			input:    `@ByteArray(\1)`,
			expected: "@ByteArray(\001)",
		},
		{
			name:     "registry",
			input:    `Software\\Foobar`,
			expected: `Software\Foobar`,
		},
		{
			name:     "registry export",
			input:    `Software\Foobar`,
			expected: `Software\Foobar`,
		},
		{
			name:     "trailing",
			input:    `abc\`,
			expected: "abc",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Unescape(tt.input)
			if got != tt.expected {
				t.Errorf("expected unescape %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestEscape(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected string
		quote    bool
		raw      bool
	}{
		{
			name:     "surrogates",
			input:    "🌎🌏🌍",
			expected: `\xd83c\xdf0e\xd83c\xdf0f\xd83c\xdf0d`,
			quote:    true,
			raw:      false,
		},
		{
			name:     "escaped",
			input:    "\n\t\r\x015\"",
			expected: `\n\t\r\0015\"`,
			quote:    true,
			raw:      false,
		},
		{
			name:     "unescaped",
			input:    "\001\"",
			expected: "\001\\\"",
			quote:    true,
			raw:      true,
		},
		{
			name:     "backslash",
			input:    `C:\Path`,
			expected: `C:\\Path`,
			quote:    true,
			raw:      false,
		},

		{
			name:     "registry export",
			input:    `Software\Foobar\🌎[]"foo"`,
			expected: `Software\\Foobar\\\xd83c\xdf0e\[\]"foo"`,
			quote:    false,
			raw:      false,
		},
		{
			name:     "regedit export",
			input:    "Software\\Foobar\\🌎[]\"foo\"\001",
			expected: "Software\\\\Foobar\\\\🌎[]\\\"foo\\\"\001",
			quote:    true,
			raw:      true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := Escape(tt.input, tt.quote, tt.raw)
			if got != tt.expected {
				t.Errorf("expected escape %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	original := "\n\t\r\x015\\\\🌎"
	escaped := Escape(original, false, false)
	unescaped := Unescape(escaped)

	if unescaped != original {
		t.Errorf("expected %q, got %q", original, unescaped)
	}
}
