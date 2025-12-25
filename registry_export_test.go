package wine

import (
	"bytes"
	"testing"
)

func TestRegistryExport(t *testing.T) {
	root := testdata()
	buf := new(bytes.Buffer) // error cannot occur here

	if err := root.exportSystem(buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if x := buf.String(); x != userExportedSys {
		t.Errorf("data unreversable")
		t.Log(x)
	}

	buf.Reset()
	t.Run("regedit", func(t *testing.T) {
		if err := root.Export(buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if x := buf.String(); x != userExported {
			t.Errorf("data unexportable")
			t.Log(x)
		}
	})

}

const userExportedSys = `WINE REGISTRY Version 2
;; All keys relative to REGISTRY\\User\\S-1-5-21-0-0-0-1000

#arch=win64

[] 1766588356
#time=1dc74e5dfeefd32
@=""
"Value A"="\"C:\\Foo\" -help"

[Foo] 1766410538
#time=1dc7347dc3ec40a
"Value B"=hex:de,ad,be,ef,00,00
"Value C"=dword:deadbeef
"Value D"=str(7):"C:\\Foo\0C:\\Bar\0"
"Value E"=str(2):"%APPDATA%\\Foo"

[Foo\\Bar] 1760553029
#time=1dc3e01c855469c
"Value F"=hex(b):ef,be,ad,de,00,00,00,00
"Value G"=str(7):"C:\\Foo\0C:\\Bar\0"
"Value H"=str(2):"%APPDATA%\\Foo"
"Value I"=hex(1):48,00,69,00,00,00

[Foo\\Bar\\Baz] 1766586874
#time=1dc74e26c24986a
"Value J"=hex(4):78,56,34,12
"Value K"=hex(5):12,34,56,78
"Value L"=hex:
"Value M"=hex(000000ff):de

[Foo\\Quz] 1766592646
#time=1dc74efdcaf516c

[Foo\\Baz] 1766592646
#time=1dc74efdcc0807c
#link
"SymbolicLinkValue"=hex(6):46,00,6f,00,6f,00,5c,00,42,00,61,00,72,00,5c,00,42,\
  00,61,00,7a,00
`

const userExported = `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER]
@=""
"Value A"="\"C:\\Foo\" -help"

[HKEY_CURRENT_USER\Foo]
"Value B"=hex:de,ad,be,ef,00,00
"Value C"=dword:deadbeef
"Value D"=hex(7):43,00,3a,00,5c,00,46,00,6f,00,6f,00,00,00,43,00,3a,00,5c,00,\
  42,00,61,00,72,00,00,00,00,00
"Value E"=hex(2):25,00,41,00,50,00,50,00,44,00,41,00,54,00,41,00,25,00,5c,00,\
  46,00,6f,00,6f,00,00,00

[HKEY_CURRENT_USER\Foo\Bar]
"Value F"=hex(b):ef,be,ad,de,00,00,00,00
"Value G"=hex(7):43,00,3a,00,5c,00,46,00,6f,00,6f,00,00,00,43,00,3a,00,5c,00,\
  42,00,61,00,72,00,00,00,00,00
"Value H"=hex(2):25,00,41,00,50,00,50,00,44,00,41,00,54,00,41,00,25,00,5c,00,\
  46,00,6f,00,6f,00,00,00
"Value I"=hex(1):48,00,69,00,00,00

[HKEY_CURRENT_USER\Foo\Bar\Baz]
"Value J"=hex(4):78,56,34,12
"Value K"=hex(5):12,34,56,78
"Value L"=hex:
"Value M"=hex(000000ff):de
`
