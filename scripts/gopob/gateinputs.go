package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

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
