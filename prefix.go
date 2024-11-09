package wine

import (
	"io"
	"os"
)

// Prefix is a representation of a Wineprefix, which is where
// Wine stores its data and is equivalent to a C:\ drive.
type Prefix struct {
	// Path to a Wine or Proton installation.
	Root string

	// Stdout and Stderr specify the descendant Prefix's Command
	// Stdout and Stderr. This is mostly reserved for logging purposes.
	// By default, they will be set to their os counterparts.
	Stderr io.Writer
	Stdout io.Writer

	dir string // Path to wineprefix.
}

// New returns a new Prefix.
//
// The given directory, an optional path to the Wineprefix, 
// must be owned by the current user, and must be an absolute path,
// otherwise running Wine will fail.
func New(dir string, root string) *Prefix {
	return &Prefix{
		Root:   root,
		Stderr: os.Stderr,
		Stdout: os.Stdout,
		dir:    dir,
	}
}

// String implements the Stringer interface, returning the directory
// of the Wineprefix.
func (p Prefix) String() string {
	return p.Dir()
}

// Dir returns the directory of the [Prefix].
func (p *Prefix) Dir() string {
	return p.dir
}
