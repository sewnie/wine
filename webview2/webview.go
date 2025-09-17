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
//
// Because.. Microsoft don't have working HTTPS certificates and I
// would like a working implementation.
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
MIIGNjCCBB6gAwIBAgITMwAAAluNg/4Rs3FOrAAAAAACWzANBgkqhkiG9w0BAQsF
ADCBhDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCldhc2hpbmd0b24xEDAOBgNVBAcT
B1JlZG1vbmQxHjAcBgNVBAoTFU1pY3Jvc29mdCBDb3Jwb3JhdGlvbjEuMCwGA1UE
AxMlTWljcm9zb2Z0IFVwZGF0ZSBTZWN1cmUgU2VydmVyIENBIDIuMTAeFw0yNTAz
MjcxODM5MTZaFw0yNjAzMjcxODM5MTZaMG4xCzAJBgNVBAYTAlVTMQswCQYDVQQI
EwJXQTEQMA4GA1UEBxMHUmVkbW9uZDESMBAGA1UEChMJTWljcm9zb2Z0MQwwCgYD
VQQLEwNEU1AxHjAcBgNVBAMTFWFwaS5jZHAubWljcm9zb2Z0LmNvbTCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBAMMcTlPzCiwN6PwiMr5AE61MGefYfv+Z
JcNJq/kplbfYKycFL//i4DJwWZefPv4wZ9I+WJNqnMgGHR9/vr44rvAhjQoL8YFf
V4elvPDFD8qnnsjxRuWOiUooMPcIYLMcqEpUgxBLMJU2P4JRXT+BRLxiwS8klBKv
ppBZKNT06y/3p9QyQ1xawQze4mVbNZEiLG5BIdmDeSDXE9uFRyuAHtdD1yywGsiZ
vyeIUSeRmRzoGko2udhGMPZbyX4QymiHEspH9G2Wh/pUVeYplHdKcINw8BBZd1+c
mZ9oRK5W0N0NTZ7cq4PS+xz/H188QUCUWWL0YvqP/VIYn/OgBUyPga0CAwEAAaOC
AbQwggGwMA4GA1UdDwEB/wQEAwIE8DATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNV
HRMBAf8EAjAAMFoGA1UdEQRTMFGCFWFwaS5jZHAubWljcm9zb2Z0LmNvbYIXKi5h
cGkuY2RwLm1pY3Jvc29mdC5jb22CHyouYmFja2VuZC5hcGkuY2RwLm1pY3Jvc29m
dC5jb20wHQYDVR0OBBYEFN66NJeD6QzTEbDPZdTeUCCCx6F8MB8GA1UdIwQYMBaA
FNLyPYR0hhtQhapd5aUHmvBH0y5pMGgGA1UdHwRhMF8wXaBboFmGV2h0dHA6Ly93
d3cubWljcm9zb2Z0LmNvbS9wa2lvcHMvY3JsL01pY3Jvc29mdCUyMFVwZGF0ZSUy
MFNlY3VyZSUyMFNlcnZlciUyMENBJTIwMi4xLmNybDB1BggrBgEFBQcBAQRpMGcw
ZQYIKwYBBQUHMAKGWWh0dHA6Ly93d3cubWljcm9zb2Z0LmNvbS9wa2lvcHMvY2Vy
dHMvTWljcm9zb2Z0JTIwVXBkYXRlJTIwU2VjdXJlJTIwU2VydmVyJTIwQ0ElMjAy
LjEuY3J0MA0GCSqGSIb3DQEBCwUAA4ICAQBRtpnQuUDcqQf2Tqs9mTvRX2oeFuRX
1pmcwGoL2A5Ja5N194QE2+RcinE2xJpGRofIhEGdn3Uyq4rUF6vzLFoTAay1AeFN
8JPouQ+UOq/bKN9J0lWt3qxtOv/neAndIrXbxHGxZLzjO+CLNPXGmVd5ZHtA0IZe
TIVlOATzdqdotSeTS0KZktfCs7KF3EkolgalMVZBd67Sacwfyqr5p6W1hIGCFdhI
uxWgVV+OpgmlHqbsEzQCGzB8sbb/ZpgSY5/eS+y9xFIJRAwg4c/LfiOtwWX6x8y6
X09FBJhPbF+SNUjF7WFuz+lNzWtiN2Sc2eVENP/nknTHIcFaMu/ca5nroUpMki3t
tWDXK/jAjnwJOGHtucBGTpo3RRHx6qZIo814G0TNu4bLhezNCjUddN2dZ3xZOps1
xmWhfVE0f5c70daZgTeuUq5vsLCB5641rSt13tv6ok9trnVsK6sWnJ1aFJYU498Y
WJppX6Z39ja3NZA8FhbcecnOvXzGhG2B8SeK2He6uPQwt3xt9RaySdCmKrnD2fz/
cnq3X8PdpXOFLCZXxhZN66dwBheyUqecgkEFbBhtVl2kX8N9n8FIOTnzgfuY0eY9
K8jo5TqJ8plg1EpZIqvBo5isnEYlWCLUiRn06WRP0LBztFtPveuInwg/C8UAmVmL
kGOd7+irF/RZMQ==
-----END CERTIFICATE-----`
