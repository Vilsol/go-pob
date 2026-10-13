package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

// TestNoTaskfileCommandRunsADeletedOriginal refuses any `task` command that still invokes a script the fold deleted.
func TestNoTaskfileCommandRunsADeletedOriginal(t *testing.T) {
	repo := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(repo, "Taskfile.yml"))
	if err != nil {
		t.Fatalf("Taskfile.yml is not readable: %v", err)
	}

	var cmds []string
	for _, ln := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if i := strings.Index(ln, " #"); i >= 0 {
			ln = ln[:i]
		}
		cmds = append(cmds, ln)
	}
	joined := strings.Join(cmds, "\n")

	found := taskOriginalRE.FindAllString(joined, -1)
	if len(found) > 0 {
		t.Errorf("%d Taskfile command(s) invoke a script the fold deleted:\n  %s\n"+
			"  Each one is a `task` target that fails the moment somebody runs it, and no suite\n"+
			"  covers it. Re-point at the gopob subcommand that replaced it.",
			len(found), strings.Join(found, "\n  "))
	}

	if len(joined) < 20_000 {
		t.Errorf("Taskfile.yml read as %d bytes — too small to be the real one, so this check "+
			"measured almost nothing", len(joined))
	}

	if !taskOriginalRE.MatchString("      - '{{.NICE}} python3 {{.HERE}}/oracle/extract_output_oracle.py {{.IN}} /tmp/x'") {
		t.Error("the pattern no longer matches the invocation that actually broke — it has been " +
			"narrowed and now reports compliance it did not measure")
	}
	if taskOriginalRE.MatchString("      - '{{.GOPOB}} extract-output-oracle {{.IN}} /tmp/x'") {
		t.Error("the pattern matches the PORTED form, so every correct target would be reported")
	}

	if difftest.OriginalsDeleted(repo) {
		t.Logf("%d command lines scanned; the originals are gone, so every match here is a live break",
			len(cmds))
	}
}

var taskOriginalRE = regexp.MustCompile(`(?:python3|bash|sh)\s+\S*(?:scripts|oracle)/[A-Za-z0-9_]+\.(?:py|sh)`)
