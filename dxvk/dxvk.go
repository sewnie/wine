// Package dxvk implements routines to install DXVK
package dxvk

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apprehensions/wine"
)

// To make DXVK usable by Wine applications, it is reccomended to use either
// the WINEDLLOVERRIDES='d3d9,d3d10core,d3d11,dxgi=n,b' variable or
// [AddOverrides] for a more permanent setting.

const Repo = "https://github.com/doitsujin/dxvk"

// Restore removes the DXVK overridden DLLs from the given wineprefix, then
// restores Wine DLLs.
func Restore(pfx *wine.Prefix) error {
	for _, dir := range []string{"syswow64", "system32"} {
		for _, dll := range []string{"d3d9", "d3d10core", "d3d11", "dxgi"} {
			p := filepath.Join(pfx.Dir(), "drive_c", "windows", dir, dll+".dll")

			if err := os.Remove(p); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return err
			}
		}
	}

	return pfx.Wine("wineboot", "-u").Run()
}

// URL returns the DXVK tarball URL for the given version.
func URL(ver string) string {
	return fmt.Sprintf("%s/releases/download/v%[2]s/dxvk-%[2]s.tar.gz", Repo, ver)
}

// Extract extracts DXVK's DLLs into the given wineprefix -
// overriding Wine's D3D DLLs, given the path to a valid DXVK tarball.
func Extract(pfx *wine.Prefix, name string) error {
	tf, err := os.Open(name)
	if err != nil {
		return err
	}
	defer tf.Close()

	zr, err := gzip.NewReader(tf)
	if err != nil {
		return err
	}
	defer zr.Close()

	tr := tar.NewReader(zr)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		if filepath.Ext(hdr.Name) != ".dll" {
			continue
		}

		var dir string
		switch filepath.Base(filepath.Dir(hdr.Name)) {
		case "x32":
			dir = "syswow64"
		case "x64":
			dir = "system32"
		default:
			continue
		}

		dst := filepath.Join(pfx.Dir(), "drive_c", "windows", dir, filepath.Base(hdr.Name))

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}

		f, err := os.Create(dst)
		if err != nil {
			return err
		}

		if _, err = io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}

		f.Close()
	}

	return nil
}
