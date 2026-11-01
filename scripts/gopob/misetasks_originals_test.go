package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

// TestNoMiseTaskRunsADeletedOriginal refuses any mise task that still invokes a script the fold deleted.
func TestNoMiseTaskRunsADeletedOriginal(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	scripts, err := filepath.Glob(filepath.Join(repo, "mise-tasks", "*"))
	if err != nil || len(scripts) == 0 {
		t.Fatalf("no scripts under mise-tasks/ (%v) — the scan would read only half the tasks", err)
	}
	sources := append([]string{filepath.Join(repo, "mise.toml")}, scripts...)

	var cmds []string
	for _, src := range sources {
		body, err := os.ReadFile(src) //nolint:gosec // the repo's own task definitions
		if err != nil {
			t.Fatalf("%s is not readable: %v", src, err)
		}
		for _, ln := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(ln), "#") {
				continue
			}
			if i := strings.Index(ln, " #"); i >= 0 {
				ln = ln[:i]
			}
			cmds = append(cmds, ln)
		}
	}
	joined := strings.Join(cmds, "\n")

	found := taskOriginalRE.FindAllString(joined, -1)
	if len(found) > 0 {
		t.Errorf("%d task command(s) invoke a script the fold deleted:\n  %s\n"+
			"  Each one is a `mise run` target that fails the moment somebody runs it, and no suite\n"+
			"  covers it. Re-point at the gopob subcommand that replaced it.",
			len(found), strings.Join(found, "\n  "))
	}

	if len(joined) < 20_000 {
		t.Errorf("the task definitions read as %d bytes — too small to be the real ones, so this check "+
			"measured almost nothing", len(joined))
	}

	if !taskOriginalRE.MatchString("  '{{vars.NICE}} python3 {{config_root}}/oracle/extract_output_oracle.py {{usage.in}} /tmp/x',") {
		t.Error("the pattern no longer matches the invocation that actually broke — it has been " +
			"narrowed and now reports compliance it did not measure")
	}
	if taskOriginalRE.MatchString("  '{{vars.GOPOB}} extract-output-oracle {{usage.in}} /tmp/x',") {
		t.Error("the pattern matches the PORTED form, so every correct target would be reported")
	}

	if difftest.OriginalsDeleted(repo) {
		t.Logf("%d command lines scanned; the originals are gone, so every match here is a live break",
			len(cmds))
	}
}

var taskOriginalRE = regexp.MustCompile(`(?:python3|bash|sh)\s+\S*(?:scripts|oracle)/[A-Za-z0-9_]+\.(?:py|sh)`)
