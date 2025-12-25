package wine

type (
	// hex(8,a,0) are removed, as they are exclusive to Windows and
	// unused by Wine.
	//
	// hex(3) removed as it is unnecessary and does not appear in any
	// real world registry data.
	//
	// Wine internally stores REG_EXPAND_SZ and REG_MULTI_SZ as str(2) and str(7)
	// and exports them as their hex(2) and hex(7) pairs respectively.
	BinaryString     []byte // hex(1): REG_SZ
	ExpandableString string // hex(2): REG_EXPAND_SZ
	DwordLE          uint32 // hex(4): REG_DWORD_LITTLE_ENDIAN
	DwordBE          uint32 // hex(5): REG_DWORD_BIG_ENDIAN
	Link             string // hex(6): REG_LINK
)

// InternalBytes represents a custom registry value type
// hex(i) where i is the Identifier. This is found in
// DEVPROP_TYPE_DEVPROPTYPE.
type InternalBytes struct {
	Identifier uint32
	Data       []byte
}

// RegistryData represents a RegistryValue's data.
//
// Known registry types and their Go types:
//   - REG_SZ = string
//   - REG_MULTI_SZ aka hex(7) = []string
//   - REG_DWORD = uint32
//   - REG_QWORD aka hex(b) = uint64
//   - REG_BINARY aka hex = []byte
//   - REG_NONE = []byte length 0
//   - REG_LINK = [Link]
//   - REG_EXPAND_SZ aka hex(2)= [ExpandableString]
//   - REG_DWORD_LITTLE_ENDIAN aka hex(4) = [DwordLE]
//   - REG_DWORD_BIG_ENDIAN aka hex(5) = [DwordBE]
//   - DEVPROP_TYPE_DEVPROPTYPE aka hex(ffff????) = [InternalByte]
type RegistryData any
