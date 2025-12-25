package wine

import (
	"encoding/binary"
	"time"
)

const offset = 116444736000000000

// Filetime is a representation of Wine's FILETIME.
//
// Routines are added and taken from golang.org/x/sys/windows package.
type Filetime int64

// Time converts Filetime to time.Time.
func (ft Filetime) Time() time.Time {
	return time.Unix(0, int64((ft-offset)*100)).UTC()
}

// FromTime converts time.Time to Filetime.
func FromTime(t time.Time) Filetime {
	return Filetime(t.UTC().UnixNano()/100 + offset)
}

// Unix returns Filetime in Unix form, truncating nanoseconds.
func (ft Filetime) Unix() int64 {
	return (int64(ft) - offset) * 100 / 1e9
}

// Bytes converts Filetime to a []byte in little endian form.
func (ft Filetime) Bytes() []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(ft))
	return buf
}

// FromBytes converts a little endian []byte to Filetime.
func FromBytes(b []byte) Filetime {
	return Filetime(int64(binary.LittleEndian.Uint64(b)))
}

// IsZero checks if Filetime represents 00:00:00 UTC, January 1, 1601.
func (ft Filetime) IsZero() bool {
	return ft == 0
}
