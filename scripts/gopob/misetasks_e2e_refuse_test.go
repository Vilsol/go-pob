package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	mtSectionRE  = regexp.MustCompile(`(?m)^\[tasks\.(e2e(?:-[a-z0-9-]+)?)\]\n((?:[^\[\n].*\n|\n)*)`)
	mtUsageRE    = regexp.MustCompile(`(?s)usage = '''(.*?)'''`)
	mtUsageLnRE  = regexp.MustCompile(`(?m)^\s*(flag|arg) "`)
	mtVariadicRE = regexp.MustCompile(`(?m)^\s*arg "\[?[a-z_]+\]?\.\.\."(.*)$`)
)

// mise passes the words after a task with no usage on to its last command, and an arg "[x]..." without
// double_dash="required" takes any flag as a value, so a mistyped flag runs the task (or the whole suite). A usage
// with a flag or a `--` arg makes mise refuse it.
func TestMiseTasksE2ERefuseUnknownFlags(t *testing.T) {
	t.Parallel()
	exempt := map[string]string{
		// e2e/serve.mjs refuses any word after its command
		"e2e-serve-stop": "serve.mjs",
		// run only by e2e/bundle.mjs
		"e2e-bundle-build": "hidden",
	}
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "mise.toml")) //nolint:gosec // the repo's own task file
	if err != nil {
		t.Fatal(err)
	}
	var open []string
	seen := map[string]bool{}
	for _, m := range mtSectionRE.FindAllStringSubmatch(string(b), -1) {
		name, body := m[1], m[2]
		// the desktop app's own harnesses, not the Playwright suite
		if strings.Contains(body, "desktop/") {
			continue
		}
		seen[name] = true
		if exempt[name] != "" {
			continue
		}
		u := mtUsageRE.FindStringSubmatch(body)
		if u == nil || !mtUsageLnRE.MatchString(u[1]) {
			open = append(open, name+": no usage flag or arg, so mise hands any word to its command")
			continue
		}
		for _, v := range mtVariadicRE.FindAllStringSubmatch(u[1], -1) {
			if !strings.Contains(v[1], `double_dash="required"`) {
				open = append(open, name+`: a variadic arg without double_dash="required" takes a mistyped flag as a value`)
			}
		}
	}
	for name := range exempt {
		if !seen[name] {
			t.Errorf("the exemption list names %q, which is no e2e task — drop it from the list", name)
		}
	}
	if len(seen) < 5 {
		t.Fatalf("found only %d e2e tasks in mise.toml; the section scan is broken", len(seen))
	}
	sort.Strings(open)
	if len(open) > 0 {
		t.Errorf("%d e2e task(s) do not refuse an unknown flag:\n  %s", len(open), strings.Join(open, "\n  "))
	}
}
