package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	manifestPath  = "scripts/testdata.json"
	installedFile = ".installed-release"
	archiveExt    = ".tar.br"
	sumsFile      = "SHA256SUMS"
)

type manifest struct {
	Repo    string            `json:"repo"`
	Release string            `json:"release"`
	Assets  map[string]string `json:"assets"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/testdatarelease pack|fetch [flags]")
		os.Exit(2)
	}
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING: run from the repository root (no go.mod here)")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "pack":
		err = cmdPack(os.Args[2:])
	case "fetch":
		err = cmdFetch(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q (pack | fetch)", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "REFUSING:", err)
		os.Exit(1)
	}
}

func readManifest() (*manifest, error) {
	b, err := os.ReadFile(filepath.FromSlash(manifestPath))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", manifestPath, err)
	}
	if m.Repo == "" || m.Release == "" || len(m.Assets) == 0 {
		return nil, fmt.Errorf("%s names no repo, release or assets", manifestPath)
	}
	return &m, nil
}

func assetDir(asset string) (string, error) {
	dir, ok := strings.CutSuffix(asset, archiveExt)
	if !ok || dir == "" || strings.ContainsAny(dir, `/\`) || dir == "." || dir == ".." {
		return "", fmt.Errorf("asset %q is not <dir>%s", asset, archiveExt)
	}
	return dir, nil
}
