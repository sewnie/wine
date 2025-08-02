// Package webview fetches and installs a Microsoft WebView version from an Upload ID.
package webview

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"strings"

	"github.com/sewnie/wine"
	"github.com/sewnie/wine/peutil"
)

var (
	ErrResourceNotFound  = errors.New("webview installer resource not found")
	ErrInstallerNotFound = errors.New("webview installer installer not found")
)

// Install runs the given WebView installer file within the Wineprefix
// with the appropiate arguments.
func Install(pfx *wine.Prefix, name string) *wine.Cmd {
	return pfx.Wine(name,
		"--msedgewebview", "--do-not-launch-msedge", "--system-level",
	)
}

// Extract uses the given ReaderAt, a file source of the Download's
// URL and extracts the WebView installer to the given dst.
func (d *Download) Extract(r io.ReaderAt, dst io.Writer) error {
	f, err := peutil.New(r)
	if err != nil {
		return err
	}
	defer f.Close()

	rs, err := f.Resources()
	if err != nil {
		return err
	}

	for _, r := range rs {
		if r.Name != "D/102/0" {
			continue
		}

		return d.extractInstaller(&r, dst)
	}

	return ErrResourceNotFound
}

func (d *Download) extractInstaller(rsrc *peutil.Resource, dst io.Writer) error {
	r := bytes.NewReader(rsrc.Data)
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if !strings.Contains(hdr.Name, d.Version) {
			continue
		}

		if _, err := io.Copy(dst, tr); err != nil {
			return err
		}

		return nil
	}

	return ErrInstallerNotFound
}
