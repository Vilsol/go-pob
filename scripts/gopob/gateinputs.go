package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// gateClosure is what a corpus gate over one package compiles and reads: every in-module package its test binary
// links, the files they embed and their testdata, the root testdata, go.mod, go.sum and the PoB pins. A change outside
// it cannot move that gate, so a docs, frontend or other-game commit keeps the gate's cache.
type gateClosure struct {
	dirs  map[string]bool
	files map[string]bool
}

var gateClosureFiles = []string{"go.mod", "go.sum", "scripts/pob_pin.json", "scripts/pob_pin_poe2.json"}

const gateClosureList = `{{if not .Standard}}D {{.Dir}}
{{range .EmbedFiles}}F {{$.Dir}}/{{.}}
{{end}}{{range .TestEmbedFiles}}F {{$.Dir}}/{{.}}
{{end}}{{range .XTestEmbedFiles}}F {{$.Dir}}/{{.}}
{{end}}{{end}}`

func newGateClosure(dirs ...string) gateClosure {
	c := gateClosure{dirs: map[string]bool{}, files: map[string]bool{}}
	for _, d := range dirs {
		c.dirs[d] = true
	}
	for _, f := range gateClosureFiles {
		c.files[f] = true
	}
	return c
}

// gateClosureOf lists pkg's closure with `go list -deps -test`; packages outside repo (the module cache) are named by
// go.sum already
func gateClosureOf(repo, pkg string) (gateClosure, error) {
	cmd := exec.CommandContext(context.Background(), "go", "list", "-deps", "-test", "-f", gateClosureList, pkg) //nolint:gosec // the caller's package
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return gateClosure{}, fmt.Errorf("go list -deps -test %s: %w", pkg, err)
	}
	c := newGateClosure()
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 3 {
			continue
		}
		rel, err := filepath.Rel(repo, line[2:])
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		if line[0] == 'D' {
			c.dirs[rel] = true
		} else {
			c.files[rel] = true
		}
	}
	if len(c.dirs) == 0 {
		return gateClosure{}, fmt.Errorf("go list -deps -test %s named no package in this repository", pkg)
	}
	return c, nil
}

func (c gateClosure) has(p string) bool {
	if c.files[p] || strings.HasPrefix(p, "testdata/") {
		return true
	}
	if strings.HasSuffix(p, ".md") {
		return false
	}
	if c.dirs[path.Dir(p)] {
		return true
	}
	if i := strings.Index(p, "/testdata/"); i >= 0 {
		return c.dirs[p[:i]]
	}
	return false
}

// key names HEAD's closure by path and blob
func (c gateClosure) key(repo string) (string, error) {
	out, err := gp2Git(repo, "ls-tree", "-r", "HEAD")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, line := range strings.Split(out, "\n") {
		if i := strings.IndexByte(line, '\t'); i >= 0 && c.has(line[i+1:]) {
			h.Write([]byte(line))
			h.Write([]byte{'\n'})
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:24], nil
}

// dirty is whether uncommitted work, untracked files included, touches the closure; the key cannot see it
func (c gateClosure) dirty(repo string) (bool, error) {
	cmd := exec.CommandContext(context.Background(), "git", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	fields := strings.Split(string(out), "\x00")
	for i := 0; i < len(fields); i++ {
		e := fields[i]
		if len(e) < 4 {
			continue
		}
		if c.has(e[3:]) {
			return true, nil
		}
		if e[0] == 'R' || e[0] == 'C' {
			i++
			if i < len(fields) && c.has(fields[i]) {
				return true, nil
			}
		}
	}
	return false, nil
}

// gateInputsSkipped is what no corpus gate reads: the frontend, the desktop shell, CI, Nix and prose. A path outside
// these is a gate input, so a new Go or data directory is covered without touching this list.
var gateInputsSkipped = []string{"frontend/", "desktop/", ".github/", "nix/"}

func gateInputsSkip(path string) bool {
	if strings.HasSuffix(path, ".md") {
		return true
	}
	for _, p := range gateInputsSkipped {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// gateInputsKey names HEAD's gate inputs: every committed file but the skipped ones, by path and blob, so a commit that
// touches only the frontend, CI or docs keeps the gate caches.
func gateInputsKey(repo string) (string, error) {
	out, err := gp2Git(repo, "ls-tree", "-r", "HEAD")
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, line := range strings.Split(out, "\n") {
		if i := strings.IndexByte(line, '\t'); i >= 0 && !gateInputsSkip(line[i+1:]) {
			h.Write([]byte(line))
			h.Write([]byte{'\n'})
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:24], nil
}
