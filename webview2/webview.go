// Package webview abstracts Edge WebView 2 updates for a Wineprefix.
package webview2

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sewnie/wine"
)

const url = "https://msedge.api.cdp.microsoft.com/api"

// Channel respresents a list of known download channels for
// Edge WebView2.
type Channel string

const (
	Stable       Channel = "msedge-stable-win"
	StableLegacy Channel = "msedge-stable-win7and8"
	Beta         Channel = "msedge-beta-win"
	Dev          Channel = "msedge-dev-win"
	Canary       Channel = "msedge-canary-win"
)

// Client is the http.Client used for requests. http.DefaultTransport
// will be used to append Microsoft's certificate.
var Client = &http.Client{}

func init() {
	t := http.DefaultTransport.(*http.Transport).Clone()
	pool, _ := x509.SystemCertPool()
	if pool == nil {
		pool = x509.NewCertPool()
	}
	pool.AppendCertsFromPEM([]byte(microsoftPEM))
	t.TLSClientConfig = &tls.Config{RootCAs: pool}
	Client.Transport = t
}

// Installed determines if the given WebView Runtime version is installed.
func Installed(pfx *wine.Prefix, version string) bool {
	_, err := os.Stat(filepath.Join(pfx.Dir(),
		"drive_c", "Program Files (x86)", "Microsoft", "EdgeWebView", "Application", version))
	return err == nil
}

// Download represents a version's available download.
type Download struct {
	URL    string `json:"Url"`
	File   string `json:"FileId"`
	Size   int64  `json:"SizeInBytes"`
	Hashes struct {
		Sha1   string `json:"Sha1"`
		Sha256 string `json:"Sha256"`
	} `json:"Hashes"`
	Delivery struct {
		CatalogID  string `json:"CatalogId"`
		Properties struct {
			IntegrityCheckInfo struct {
				PiecesHashFileURL string `json:"PiecesHashFileUrl"`
				HashOfHashes      string `json:"HashOfHashes"`
			} `json:"IntegrityCheckInfo"`
		} `json:"Properties"`
	} `json:"DeliveryOptimization"`
}

// Path returns the executable path of the Download URL as represented
// in the Wineprefix.
//
// It is the user's responsibility to ensure this exists if using [Download.Install],
// by fetching the [Download.URL] to the path returned here.
func (d *Download) Path(pfx *wine.Prefix) string {
	id := "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
	v, _ := d.Version()
	return filepath.Join(pfx.Dir(),
		"drive_c", "Program Files (x86)", "Microsoft", "EdgeUpdate", "Download", id, v, d.File)
}

// Install runs the downloaded executable with arguments to install it onto the Wineprefix.
//
// To ensure WebView2 runs correctly within the Wineprefix, a windows version override is installed
// by default if the Wineprefix is not Proton, since the override is installed in Proton by default.
func (d *Download) Install(pfx *wine.Prefix) error {
	if !pfx.IsProton() {
		if err := pfx.RegistryAdd(`HKCU\Software\Wine\AppDefaults\msedgewebview2.exe`, "Version", "win7"); err != nil {
			return fmt.Errorf("version set: %w", err)
		}
	}

	return pfx.Wine(d.Path(pfx),
		"--msedgewebview", "--do-not-launch-msedge", "--system-level",
	).Run()
}

// Version returns the DownloadInfo's runtime and Edge version.
// If the returned Edge version is empty, this DownloadInfo is a Runtime.
func (d *Download) Version() (string, string) {
	name := strings.Split(strings.TrimSuffix(d.File, ".exe"), "_")
	switch len(name) {
	case 3:
		return name[2], ""
	case 4:
		return name[2], name[3]
	default:
		return "unknown", "unknown"
	}
}

// Latest returns the latest version of the given WebView download channel.
//
// arch should be one of "x86", "x64", "ARM64".
func (c Channel) Latest(arch string) (string, error) {
	// code 4006: Action must be 'select'
	r := strings.NewReader(`{"targetingAttributes":{"Updater":"MicrosoftEdgeUpdate"}}`)
	resp, err := Client.Post(fmt.Sprintf(
		"%s/v1.1/contents/Browser/namespaces/Default/names/%s-%s/versions/latest?action=select",
		url, c, arch), "application/json", r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("webview: bad status")
	}

	data := struct {
		ContentID struct {
			Namespace string `json:"Namespace"`
			Name      string `json:"Name"`
			Version   string `json:"Version"`
		} `json:"ContentId"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.ContentID.Version, nil
}

// Download fetches the downloads for the given WebView download channel and version.
// The downloads that are returned consist of Edge versions and a single Runtime.
//
// arch should be one of "x86", "x64", "ARM64".
func (c Channel) Downloads(version, arch string) ([]Download, error) {
	resp, err := Client.Post(fmt.Sprintf(
		"%s/v1.1/contents/Browser/namespaces/Default/names/%s-%s/versions/%s/files?action=GenerateDownloadInfo",
		url, c, arch, version), "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("webview: bad status")
	}

	var data []Download
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// Runtime fetches the download of the Edge WebView2 Runtime for the given
// WebView download channel and version. If it could not be found, an
// error will be returned.
//
// Runtime wraps around [Channel.Download].
//
// arch should be one of "x86", "x64", "ARM64".
func (c Channel) Runtime(version, arch string) (*Download, error) {
	downloads, err := c.Downloads(version, arch)
	if err != nil {
		return nil, err
	}

	for _, d := range downloads {
		if _, e := d.Version(); e == "" {
			return &d, nil
		}
	}

	return nil, errors.New("webview: runtime missing")
}

const microsoftPEM = `-----BEGIN CERTIFICATE-----
MIIF7TCCA9WgAwIBAgIQP4vItfyfspZDtWnWbELhRDANBgkqhkiG9w0BAQsFADCB
iDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCldhc2hpbmd0b24xEDAOBgNVBAcTB1Jl
ZG1vbmQxHjAcBgNVBAoTFU1pY3Jvc29mdCBDb3Jwb3JhdGlvbjEyMDAGA1UEAxMp
TWljcm9zb2Z0IFJvb3QgQ2VydGlmaWNhdGUgQXV0aG9yaXR5IDIwMTEwHhcNMTEw
MzIyMjIwNTI4WhcNMzYwMzIyMjIxMzA0WjCBiDELMAkGA1UEBhMCVVMxEzARBgNV
BAgTCldhc2hpbmd0b24xEDAOBgNVBAcTB1JlZG1vbmQxHjAcBgNVBAoTFU1pY3Jv
c29mdCBDb3Jwb3JhdGlvbjEyMDAGA1UEAxMpTWljcm9zb2Z0IFJvb3QgQ2VydGlm
aWNhdGUgQXV0aG9yaXR5IDIwMTEwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIK
AoICAQCygEGqNThNE3IyaCJNuLLx/9VSvGzH9dJKjDbu0cJcfoyKrq8TKG/Ac+M6
ztAlqFo6be+ouFmrEyNozQwph9FvgFyPRH9dkAFSWKxRxV8qh9zc2AodwQO5e7BW
6KPeZGHCnvjzfLnsDbVU/ky2ZU+I8JxImQxCCwl8MVkXeQZ4KI2JOkwDJb5xalwL
54RgpJki49KvhKSn+9GY7Qyp3pSJ4Q6g3MDOmT3qCFK7VnnkH4S6Hri0xElcTzFL
h93dBWcmmYDgcRGjuKVB4qRTufcyKYMME782XgSzS0NHL2vikR7TmE/dQgfI6B0S
/Jmpaz6SfsjWaTr8ZL22CZ3K/QwLopt3YEsDlKQwaRLWQi3BQUzK3Kr9j1uDRprZ
/LHR47PJf0h6zSTwQY9cdNCssBAgBkm3xy0hyFfj0IbzA2j70M5xwYmZSmQBbP3s
MJHPQTySx+W6hh1hhMdfgzlirrSSL0fzC/hV66AfWdC7dJse0Hbm8ukG1xDo+mTe
acY1logC8Ea4PyeZb8txiSk190gWAjWP1Xl8TQLPX+uKg09FcYj5qQ1OcunCnAfP
SRtOBA5jUYxe2ADBVSy2xuDCZU7JNDn1nLPEfuhhbhNfFcRf2X7tHc7uROzLLoax
7Dj2cO2rXBPB2Q8Nx4CyVe0096yb5MPa50c8prWPMd/FS6/r8QIDAQABo1EwTzAL
BgNVHQ8EBAMCAYYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUci06AjGQQ7kU
BU7h6qfHMdEjiTQwEAYJKwYBBAGCNxUBBAMCAQAwDQYJKoZIhvcNAQELBQADggIB
AH9yzw+3xRXbm8BJyiZb/p4T5tPw0tuXX/JLP02zrhmu7deXoKzvqTqjwkGw5biR
nhOBJAPmCf0/V0A5ISRW0RAvS0CpNoZLtFNXmvvxfomPEf4YbFGq6O0JlbXlccmh
6Yd1phV/yX43VF50k8XDZ8wNT2uoFwxtCJJ+i92Bqi1wIcM9BhS7vyRep4TXPw8h
Ir1LAAbblxzYXtTFC1yHblCk6MM4pPvLLMWSZpuFXst6bJN8gClYW1e1QGm6CHmm
ZGIVnYeWRbVmIyADixxzoNOieTPgUFmG2y/lAiXqcyqfABTINseSO+lOAOzYVgm5
M0kS0lQLAausR7aRKX1MtHWAUgHoyoL2n8ysnI8X6i8msKtyrAv+nlEex0NVZ09R
s1fWtuzuUrc66U7h14GIvE+OdbtLqPA1qibUZ2dJsnBMO5PcHd94kIZysjik0dyS
TclY6ysSXNQ7roxrsIPlAT/4CTL2kzU0Iq/dNw13CYArzUgA8YyZGUcFAenRv9FO
0OYoQzeZpApKCNmacXPSqs0xE2N2oTdvkjgefRI8ZjLny23h/FKJ3crWZgWalmG+
oijHHKOnNlA8OqTfSm7mhzvO6/DggTedEzxSjr25HTTGHdUKaj2YKXCMiSrRq4IQ
SB/c9O+lxbtVGjhjhE63bK2VVOxlIhBJF7jAHscPrFRH
-----END CERTIFICATE-----`
