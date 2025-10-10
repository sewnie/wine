// Package dxvk manages DXVK for a Wineprefix.
//
// To make DXVK usable by Wine applications, it is reccomended to use either
// [Variable] for runtime setting or [AddOverrides] for a more permanent setting.
package dxvk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sewnie/wine"
	"github.com/sewnie/wine/peutil"
)

const Repo = "https://github.com/doitsujin/dxvk"

// EnvOverride appends DXVK DLL overrides to the given Wineprefix's
// environment variables.
func EnvOverride(pfx *wine.Prefix, enabled bool) {
	name := "WINEDLLOVERRIDES"
	val := "d3d9,d3d10core,d3d11,dxgi="
	if enabled {
		val += "native"
	} else {
		val += "builtin"
	}

	for i, env := range pfx.Env {
		if !strings.HasPrefix(env, name) {
			continue
		}

		pfx.Env[i] += ";" + val
		return
	}

	pfx.Env = append(pfx.Env, name+"="+val)
}

// Restore restores Direct3D DLLs, which were overwritten by DXVK, in the wineprefix.
func Restore(pfx *wine.Prefix) error {
	dirs := []string{"syswow64", "system32"}
	names := []string{"d3d8", "d3d9", "d3d10core", "d3d11", "dxgi"}

	for _, dir := range dirs {
		for _, name := range names {
			dll := filepath.Join(pfx.Dir(), "drive_c", "windows", dir, name+".dll")

			if err := os.Remove(dll); err != nil {
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

// Version returns the DXVK version of the system32 d3d11 DLL installed
// in the wineprefix. The 'd3d11' DLL is chosen as it is one of
// the only DXVK DLLs that contain versioning.
//
// If other DLLs such as d3d8 are needed to track, it is reccomended
// to store the installed version of DXVK prior to [Extract].
func Version(pfx *wine.Prefix) (string, error) {
	return dllVersion(filepath.Join(
		pfx.Dir(), "drive_c", "windows", "system32", "d3d11.dll"))
}

// Only valid for d3d9, d3d11 & dxgi
func dllVersion(name string) (string, error) {
	f, err := peutil.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()

	for _, s := range f.Sections {
		if s.Name != ".rdata" {
			continue
		}
		b, err := s.Data()
		if err != nil {
			log.Fatal(err)
		}
		head := []byte("DXVK: \x00v")
		start := bytes.Index(b, head)
		if start < 0 {
			break
		}
		verStart := start + len(head)
		verEnd := bytes.IndexByte(b[verStart:], 0)
		if verEnd < 0 {
			break
		}
		return string(b[verStart : verStart+verEnd]), nil
	}

	return "", nil
}

// Extract installs the DXVK DLLs by seeking to the start of
// tarball and extracting the gzipped contents onto the given
// wineprefix. Extract will override Wine DLLs; to use it,
// you will have to add DLL overrides via [EnvOverride].
func Extract(pfx *wine.Prefix, tarball io.ReadSeeker) error {
	if _, err := tarball.Seek(0, io.SeekStart); err != nil {
		return err
	}

	zr, err := gzip.NewReader(tarball)
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

		log.Println("dxvk: Installing", dst)

		if _, err = io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}

		f.Close()
	}

	return nil
}
