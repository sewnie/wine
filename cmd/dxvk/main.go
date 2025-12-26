package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sewnie/wine"
	"github.com/sewnie/wine/dxvk"
)

func main() {
	var version string
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] install|uninstall\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&version, "ver", "2.7.1", "dxvk version to install")
	flag.Parse()

	pfx := wine.New(os.Getenv("WINEPREFIX"), "")
	pfx.Env = append(pfx.Env, "WINEDLLOVERRIDES=winemenubuilder.exe=")
	if !pfx.Exists() {
		log.Println("Initializing Wineprefix")

		err := pfx.Init()
		if err != nil {
			log.Fatalln("failed to initialize:", err)
		}
	}

	v, err := dxvk.Version(pfx)
	if err != nil {
		log.Fatal("failed to find installed dxvk:", err)
	}
	log.Printf("Installed DXVK version: %v", v)

	switch flag.Arg(0) {
	case "install":
		err := installDXVK(pfx, version)
		if err != nil {
			log.Fatal(err)
		}
	case "uninstall":
		err := dxvk.Restore(pfx)
		if err != nil {
			log.Fatal(err)
		}
	case "env":
		log.Println(pfx.Env)
		dxvk.EnvOverride(pfx, true)
		log.Println(pfx.Env)
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func installDXVK(pfx *wine.Prefix, version string) error {
	out, err := os.CreateTemp("", "dxvk.*.tar.gz")
	if err != nil {
		return fmt.Errorf("tempfile: %w", err)
	}
	defer os.Remove(out.Name())

	url := dxvk.URL(version)
	log.Println("Fetching URL:", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	return dxvk.Extract(pfx, out)
}
