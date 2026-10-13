package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var uDispatchAnchor = "\tswitch os.Args[1] {"

var (
	uCaseRE      = regexp.MustCompile(`(?m)^\tcase "([a-z0-9-]+)":`)
	uPreSwitchRE = regexp.MustCompile(`os\.Args\[1\] == "([a-z0-9-]+)"`)
	uEntryRE     = regexp.MustCompile(`(?m)^  ([a-z0-9-]+)`)
)

var uNotSubcommands = map[string]bool{"-h": true, "--help": true, "help": true}

func uDispatchCases(t *testing.T) []string {
	t.Helper()
	repo := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(repo, "scripts", "gopob", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	i := strings.Index(src, uDispatchAnchor)
	if i < 0 {
		t.Fatalf("the dispatch anchor %q is gone from main.go — the scan would silently read the "+
			"wrong switch, or none", uDispatchAnchor)
	}
	end := strings.Index(src[i:], "\n}\n")
	if end < 0 {
		t.Fatal("main()'s closing brace is gone — the scan would run past the dispatch switch")
	}
	dispatch := src[i : i+end]
	var out []string
	for _, m := range uPreSwitchRE.FindAllStringSubmatch(src[:i], -1) {
		out = append(out, m[1])
	}
	for _, m := range uCaseRE.FindAllStringSubmatch(dispatch, -1) {
		if uNotSubcommands[m[1]] {
			continue
		}
		out = append(out, m[1])
	}
	if len(out) < 30 {
		t.Fatalf("only %d dispatch cases found in main.go — the scan stopped matching, and an "+
			"empty scan is a check that passes without looking", len(out))
	}
	return out
}

func TestEverySubcommandAppearsInTheUsageText(t *testing.T) {
	var missing []string
	for _, name := range uDispatchCases(t) {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(name) + `( |$)`).MatchString(usage) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d subcommand(s) dispatch but are absent from the usage text, so nothing but "+
			"main.go can find them:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// TestEveryUsageEntryNamesALiveSubcommand: THE REVERSE DIRECTION IS THE `-run NoSuchTest` CLASS.
func TestEveryUsageEntryNamesALiveSubcommand(t *testing.T) {
	live := map[string]bool{}
	for _, name := range uDispatchCases(t) {
		live[name] = true
	}
	documentedElsewhere := map[string]bool{
		"gate-versions": true,
	}
	var dead []string
	for _, m := range uEntryRE.FindAllStringSubmatch(usage, -1) {
		if live[m[1]] || documentedElsewhere[m[1]] {
			continue
		}
		dead = append(dead, m[1])
	}
	if len(dead) > 0 {
		t.Errorf("%d usage entr(ies) name a subcommand that does not dispatch:\n  %s",
			len(dead), strings.Join(dead, "\n  "))
	}
}
