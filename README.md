# wine
[pkg.go.dev]:     https://pkg.go.dev/github.com/apprehensions/wine
[pkg.go.dev_img]: https://img.shields.io/badge/%E2%80%8B-reference-007d9c?logo=go&logoColor=white&style=flat-square

[![Godoc Reference][pkg.go.dev_img]][pkg.go.dev]

A Go package for managing a Wineprefix and running Wine.

### Example application client
```go
package main

import (
	"os"
	"log"
	"io"
	"flag"
	"path/filepath"

	"github.com/apprehensions/wine"
)

func main() {
	root := flag.String("root", "", "Path to a wine install")
	dir := flag.String("dir",
		filepath.Join(os.Getenv("HOME"), ".wine"), "Path to a wineprefix")
	flag.Parse()

	logFile, err := os.CreateTemp(os.TempDir(), "wine-stderr.*.log")
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.Println("Log file at:", logFile.Name())

	pfx := wine.New(*dir, *root)
	pfx.Stderr = io.MultiWriter(os.Stderr, logFile)
	if err := pfx.Init(); err != nil {
		log.Fatal(err)
	}

	if err := pfx.SetDPI(96); err != nil {
		log.Fatal(err)
	}

	appData, err := pfx.AppDataDir()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("User's AppData directory:", appData)

	err = pfx.RegistryAdd(`HKEY_CURRENT_USER\Software\Wine\Explorer\Desktops`,
		"Default", wine.REG_SZ, "1920x1080")
    if err != nil {
		log.Fatal(err)
	}

	err = pfx.RegistryDelete(`HKEY_CURRENT_USER\Software\Wine\Explorer\Desktops`,
		"Default")
	if err != nil {
		log.Fatal(err)
	}

	wineVer := pfx.Version()
	log.Println("Wine version:", wineVer)

	_ = pfx.Kill()
}
```
