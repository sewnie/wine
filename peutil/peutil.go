package peutil

import (
	"debug/pe"
	"io"
)

// File represents a PE file. It wraps a pe.File to provide access to more
// headers and elements.
type File struct {
	*pe.File
}

// Open opens the named PE file
func Open(name string) (*File, error) {
	p, err := pe.Open(name)
	return &File{p}, err
}

// New initializes a File from a ReaderAt
func New(r io.ReaderAt) (*File, error) {
	p, err := pe.NewFile(r)
	return &File{p}, err
}
