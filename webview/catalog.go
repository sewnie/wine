package webview

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

const catalog = `https://www.catalog.update.microsoft.com/DownloadDialog.aspx`

var ErrExtractFail = errors.New("download information extraction failed")

var info = regexp.MustCompile(`(enTitle.*=.*'*\(Build ([^)]+)\)'|files\[0\]\.url.*=.*'([^']+)');`)

type Download struct {
	ID      string
	Version string
	URL     string
}

// GetDownload retrieves a WebView2 installer from the Microsoft Update Catalog
// for the given updateID.
func GetDownload(updateID string) (*Download, error) {
	data := url.Values{}
	data.Set("updateIDs",
		`[{"size":0,"languages":"","uidInfo":"`+updateID+`","updateID":"`+updateID+`"}]`)

	resp, err := http.PostForm(catalog, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := info.FindAllStringSubmatch(string(body), -1)

	if len(m) != 2 || len(m[0]) != 4 || len(m[1]) != 4 {
		return nil, ErrExtractFail
	}

	return &Download{
		ID:      updateID,
		Version: m[0][2],
		URL:     m[1][3],
	}, nil
}
