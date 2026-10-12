package parsecache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var fingerprintSources = []string{"builds", "builds/parsecache", "pob"}

var ParserFingerprint = sync.OnceValues(func() (string, error) {
	root, err := checkoutRoot()
	if err != nil {
		return "", err
	}

	h := sha256.New()
	fmt.Fprintf(h, "parsecache/v%d\n", formatVersion)
	for _, pkg := range fingerprintSources {
		dir := filepath.Join(root, pkg)
		ents, err := os.ReadDir(dir)
		if err != nil {
			return "", fmt.Errorf("parsecache: fingerprint: %w", err)
		}
		names := make([]string, 0, len(ents))
		for _, e := range ents {
			n := e.Name()
			if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
				continue
			}
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			//nolint:gosec // G703: a .go file inside this checkout, named by os.ReadDir.
			b, err := os.ReadFile(filepath.Join(dir, n))
			if err != nil {
				return "", fmt.Errorf("parsecache: fingerprint: %w", err)
			}
			fmt.Fprintf(h, "%s/%s %d\n", pkg, n, len(b))
			h.Write(b)
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:32], nil
})

func checkoutRoot() (string, error) {
	if r := os.Getenv("GOPOB_ROOT"); r != "" {
		if declaresModule(r) {
			return r, nil
		}
		return "", fmt.Errorf("parsecache: GOPOB_ROOT does not contain a go.mod declaring module github.com/Vilsol/go-pob")
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("parsecache: no working directory: %w", err)
	}
	for {
		if declaresModule(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("parsecache: no go.mod declaring module github.com/Vilsol/go-pob at or above the working directory")
		}
		dir = parent
	}
}

func declaresModule(dir string) bool {
	//nolint:gosec // G703: a go.mod inside the directory being probed for the checkout root.
	b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "module github.com/Vilsol/go-pob" {
			return true
		}
	}
	return false
}
