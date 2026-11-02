package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var wsSources = regexp.MustCompile(`(?m)^\[task_templates\.wasm-sources\]\nsources = \[([^\]]*)\]`)

// The wasm builds are skipped when nothing in wasm-sources changed, so a package the wasm compiles that the list
// misses would serve a stale wasm.
func TestWasmSourcesCoverEveryPackageTheWasmCompiles(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "mise.toml")) //nolint:gosec // the repo's own task file
	if err != nil {
		t.Fatal(err)
	}
	m := wsSources.FindSubmatch(b)
	if m == nil {
		t.Fatal("mise.toml has no [task_templates.wasm-sources] sources list")
	}
	parts := strings.Split(string(m[1]), ",")
	globs := make([]string, 0, len(parts))
	for _, g := range parts {
		globs = append(globs, strings.Trim(strings.TrimSpace(g), `"`))
	}
	for _, need := range []string{"go.mod", "go.sum"} {
		if !strings.Contains(string(m[1]), `"`+need+`"`) {
			t.Errorf("wasm-sources leaves out %s; a dependency bump would serve the old wasm", need)
		}
	}
	cmd := exec.CommandContext(t.Context(), "go", "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{range .EmbedFiles}} {{.}}{{end}}{{end}}", "./wasm", "./wasm2")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, ln := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(ln)
		if len(f) == 0 || !strings.HasPrefix(f[0], root+string(filepath.Separator)) {
			continue
		}
		seen++
		rel, _ := filepath.Rel(root, f[0])
		if len(f) > 1 {
			t.Errorf("%s embeds %v, which the .go-only globs do not watch", rel, f[1:])
		}
		top := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		covered := false
		for _, g := range globs {
			if g == top+"/**/*.go" {
				covered = true
			}
		}
		if !covered {
			t.Errorf("the wasm compiles %s, but wasm-sources has no %s/**/*.go", rel, top)
		}
	}
	if seen == 0 {
		t.Fatal("go list named no package of this module; the check covered nothing")
	}
}
