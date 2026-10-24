package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const triagePy = "scripts/triage_drops.py"

const tiedRows = 24

type vrec struct {
	Parsed *bool    `json:"parsed,omitempty"`
	Extra  *string  `json:"extra,omitempty"`
	Names  []string `json:"names"`
	Line   *string  `json:"line,omitempty"`
}

type crec struct {
	Name *string `json:"name,omitempty"`
	Raw  *string `json:"raw,omitempty"`
}

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }

func triageVerdicts() []vrec {
	yes := bp(true)
	v := []vrec{
		{Parsed: yes, Names: []string{"HasConsumerA"}, Line: sp("WEIGHTED LINE")},
		{Parsed: yes, Names: []string{"HasConsumerB", "HasConsumerC"}, Line: sp("TWO CONSUMERS")},
		{Parsed: yes, Names: []string{"OnlyInParser"}, Line: sp("PARSER ONLY")},
		{Parsed: yes, Names: []string{"OnlyInTest"}, Line: sp("TEST ONLY")},
		{Parsed: bp(false), Names: []string{"HasConsumerA"}, Line: sp("NOT PARSED")},
		{Parsed: yes, Extra: sp("  leftover  "), Names: []string{"HasConsumerA"}, Line: sp("HAS EXTRA")},
		{Parsed: yes, Names: []string{}, Line: sp("NO NAMES")},
		{Parsed: yes, Names: []string{"HasConsumerA"}, Line: sp(strings.Repeat("’", 90))},
		{Parsed: yes, Names: []string{"Ünïcode"}, Line: sp("SHORT UNICODE NAME")},
		{Parsed: yes, Names: []string{"HasConsumerA", "éééNameThatIsQuiteLongIndeed",
			"AndAnotherLongOne"}, Line: sp("WIDE NAMES")},
	}
	out := make([]vrec, 0, len(v)+tiedRows)
	for i := range tiedRows {
		out = append(out, vrec{Parsed: yes, Names: []string{"HasConsumerA"},
			Line: sp(fmt.Sprintf("TIED %02d", i))})
		if i == tiedRows/2 {
			out = append(out, v...)
		}
	}
	return out
}

func triageCases() []crec {
	return []crec{
		{Name: sp("x|7|y"), Raw: sp("  first\nWEIGHTED LINE  ")},
		{Name: sp("x| 5 |y"), Raw: sp("TWO CONSUMERS")},
		{Name: sp("nopipe")},
		{Name: sp("x|notanumber|y"), Raw: sp("PARSER ONLY")},
		{Name: sp("x|1|y"), Raw: sp("TEST ONLY")},
		{Name: sp("x|9|y"), Raw: sp("TEST ONLY")},
	}
}

func writeJSON(t *testing.T, p string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func triageSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	base := t.TempDir()
	py := difftest.ReadOriginal(t, repo, triagePy)
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			triagePy:                    string(py),
			"calculator/consumer.go":    "package calculator\n\nvar a = \"HasConsumerA\"\n",
			"calculator/mod_parser.go":  "package calculator\n\nvar b = \"OnlyInParser\"\n",
			"calculator/helper_test.go": "package calculator\n\nvar c = \"OnlyInTest\"\n",
			"data/d.go":                 "package data\n\nvar d = \"HasConsumerB\"\n",
			"moddb/m.go":                "package moddb\n\nvar e = \"HasConsumerC\"\n",
			"moddb/u.go":                "package moddb\n\nvar f = \"Ünïcode\"\n",
		}
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		for rel, v := range map[string]any{
			"verdict.json": triageVerdicts(),
			"cases.json":   triageCases(),
		} {
			b, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(filepath.Join(sb, rel), b, 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func rewriteJSON(sb, rel string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	if err := os.WriteFile(filepath.Join(sb, rel), b, 0o600); err != nil {
		return fmt.Errorf("fixture: %w", err)
	}
	return nil
}

func TestTriageDropsPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := triageSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, triagePy)

	args := []string{"verdict.json", "cases.json"}
	spec := difftest.Spec{
		Tool:         "triage-drops",
		PythonScript: triagePy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"triage-drops"}, args...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 9,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "REFUSING: usage: triage_drops.py",
				Break: func(string) error { return nil }},

			{Name: "bad-top", Refusal: "REFUSING: top must be a non-negative integer, not abc",
				Break: func(string) error { return nil }},

			{Name: "verdict-missing", Refusal: "verdict.json is not readable:",
				Break: func(sb string) error { return os.Remove(filepath.Join(sb, "verdict.json")) }},
			{Name: "verdict-unreadable", Refusal: "verdict.json is not readable:",
				Break: func(sb string) error {
					return os.Chmod(filepath.Join(sb, "verdict.json"), 0o000)
				}},

			{Name: "verdict-bad-json", Refusal: "verdict.json is not valid JSON",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, "verdict.json"), []byte("{not json"), 0o600)
				}},

			{Name: "verdict-empty", Refusal: "REFUSING: verdict file is empty",
				Break: func(sb string) error { return rewriteJSON(sb, "verdict.json", []vrec{}) }},

			{Name: "verdict-no-names-key", Refusal: "no record carries `names`",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, "verdict.json"),
						[]byte(`[{"parsed":true,"line":"x"}]`), 0o600)
				}},

			{Name: "case-no-name", Refusal: "cases.json record 0 has no `name`",
				Break: func(sb string) error { return rewriteJSON(sb, "cases.json", []crec{{}}) }},

			{Name: "case-no-raw", Refusal: "cases.json record 0 has no `raw`",
				Break: func(sb string) error {
					return rewriteJSON(sb, "cases.json", []crec{{Name: sp("x|1|y")}})
				}},

			{Name: "verdict-no-line", Refusal: "verdict.json record 0 has no `line`",
				Break: func(sb string) error {
					return os.WriteFile(filepath.Join(sb, "verdict.json"),
						[]byte(`[{"parsed":true,"names":["HasConsumerA"]}]`), 0o600)
				}},

			{Name: "top-zero", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if f.Name == "verdict-unreadable" && os.Geteuid() == 0 {
				t.Skip("running as root, so a mode-000 file is still readable and this fixture " +
					"cannot fail")
			}
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs, s.GoArgs = nil, []string{"triage-drops"}
			case "bad-top":
				s.PythonArgs = append(append([]string{}, args...), "abc")
				s.GoArgs = append([]string{"triage-drops"}, s.PythonArgs...)
			case "top-zero":
				s.PythonArgs = append(append([]string{}, args...), "0")
				s.GoArgs = append([]string{"triage-drops"}, s.PythonArgs...)
			}
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"cases.json", "verdict.json"}, mk)
}

func TestTriageDropsPythonIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := triageSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "triage-drops",
		PythonScript: triagePy,
		PythonArgs:   []string{"verdict.json", "cases.json"},
	}, mk, []string{"0", "12345", "999983"})
}

// TestHasConsumerAgreesWithRealGrep: THE ONE PLACE THE PORT IS NOT A TRANSLITERATION, SO IT OWES A CONTROL.
func TestHasConsumerAgreesWithRealGrep(t *testing.T) {
	repo := repoRoot(t)
	t.Chdir(repo)

	src, err := os.ReadFile("calculator/mod_parser.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	var names []string
	for _, m := range regexp.MustCompile(`"([A-Z][A-Za-z]{6,30})"`).FindAllStringSubmatch(string(src), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	if len(names) < 20 {
		t.Fatalf("only %d candidate names; the sample is too small to mean anything", len(names))
	}
	step := len(names) / 20
	var sample []string
	for i := 0; i < len(names) && len(sample) < 20; i += step {
		sample = append(sample, names[i])
	}

	yes, no := 0, 0
	for _, n := range sample {
		want := grepHasConsumer(t, n)
		if got := hasConsumer(n); got != want {
			t.Errorf("%q: native search says %v, grep says %v", n, got, want)
		}
		if want {
			yes++
		} else {
			no++
		}
	}
	if yes == 0 || no == 0 {
		t.Fatalf("VACUOUS: %d names, all answered %v — the sample cannot distinguish the "+
			"implementation from a constant", len(sample), yes > 0)
	}
	t.Logf("%d names agreed: %d with a consumer, %d without", len(sample), yes, no)
}

func grepHasConsumer(t *testing.T, name string) bool {
	t.Helper()
	//nolint:gosec // the "taint" is a Go identifier this test just read out of our own source,
	out, _ := exec.CommandContext(t.Context(), "grep", "-rl", `"`+name+`"`,
		"--include=*.go", "calculator/", "data/", "moddb/").Output()
	for _, f := range strings.Split(string(out), "\n") {
		if f == "" || strings.HasSuffix(f, "_test.go") ||
			strings.HasSuffix(f, "calculator/mod_parser.go") {
			continue
		}
		return true
	}
	return false
}
