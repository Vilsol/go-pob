package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const scPy = "scripts/source_census.py"

const scN = 14

func scPoBLife(i int) float64 { return 100.0 + float64(i) }

func scPoBTiny(i int) float64 { return 0.0002 * float64(i+1) }

func scOurTiny(i int) float64 {
	switch i {
	case 6:
		return scPoBTiny(6) + 0.00005
	case 7:
		return scPoBTiny(7) + 0.0003
	}
	return scPoBTiny(i)
}

func scOurLife(i int) float64 {
	switch {
	case i < 6:
		return 100.0 + float64(i)
	case i < 9:
		return (100.0 + float64(i)) * 2
	case i < 12:
		return (100.0 + float64(i)) * 0.5
	case i == 12:
		return 0.0
	}
	return 100.0 + float64(i)
}

func scGz(t *testing.T, body string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func scRunDump() string {
	var b []byte
	add := func(f string, a ...any) { b = append(b, fmt.Sprintf(f, a...)...) }
	add("V\tdataVersion\t3.29\n")
	add("V\tcorpusDir\ttestdata/corpus-3.29  \n")
	for i := range scN {
		add("B\tb%02d.xml\tok\n", i)
		switch i {
		case 3:
			add("S\tLife\t1_03.0\n")
		case 4:
			add("S\tLife\t  104.0  \n")
		default:
			add("S\tLife\t%s\n", strconv.FormatFloat(scOurLife(i), 'g', -1, 64))
		}
		add("S\tArmour\t%s\n", strconv.FormatFloat(50.0+float64(i), 'g', -1, 64))
		add("S\tTiny\t%s\n", strconv.FormatFloat(scOurTiny(i), 'g', -1, 64))
	}
	return string(b)
}

func scBlobBody() string {
	var b []byte
	for i := range scN {
		out := map[string]any{"Armour": 50.0 + float64(i), "Tiny": scPoBTiny(i)}
		if i != 13 {
			out["Life"] = scPoBLife(i)
		}
		rec := map[string]any{
			"build": fmt.Sprintf("b%02d.xml", i), "treeSources": 9, "output": out,
			"minionOutput": map[string]any{"MinionLife": 500.0 + float64(i)},
			"enemyOutput":  map[string]any{"EnemyLife": 900.0 + float64(i)},
		}
		j, _ := json.Marshal(rec)
		b = append(b, j...)
		b = append(b, '\n')
	}
	return string(b)
}

func scStatSources(i int) string {
	failing := 6 <= i && i < 12
	base := []any{
		map[string]any{"source": "Base"},
		map[string]any{"source": "Tree:18769 (Written in Blood)"},
		map[string]any{"source": "Item:538:Loath Ward, Silken Hood"},
		map[string]any{"source": "Strength"},
	}
	inc := []any{map[string]any{"source": fmt.Sprintf("Tree:%d (Node %d)", 1000+i, i)}}
	if failing {
		inc = append(inc, map[string]any{"source": fmt.Sprintf("Tree:%d (Other %d)", 2000+i, i)})
	}
	life := map[string]any{
		"value":    scPoBLife(i),
		"BASE":     base,
		"INC":      inc,
		"OVERRIDE": "not a list",
	}
	if i == 7 {
		life["value"] = []any{map[string]any{"source": "PhantomTotal"}}
	}
	if failing {
		life["MORE"] = []any{
			map[string]any{"source": "Skill:Discipline"},
			map[string]any{"source": "Spectre:Host Chieftain"},
			map[string]any{"source": "Item:99:Kaom's Heart"},
			map[string]any{"source": "Tree:31337 (Blood Magic)"},
			map[string]any{"kind": "unlabelled"},
			map[string]any{"source": "Flask:Divine Life"},
		}
	}
	j, _ := json.Marshal(map[string]any{
		"statSources": map[string]any{
			"Life":   life,
			"Armour": map[string]any{"value": 50.0 + float64(i), "BASE": base},
		},
	})
	return string(j)
}

func scSandbox(t *testing.T, repo string, extra map[string]string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, scPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			scPy:                   string(py),
			"ours.txt.gz":          scGz(t, scRunDump()),
			"blob/output.jsonl.gz": scGz(t, scBlobBody()),
			"notes/readme.txt":     "a directory that is not a dump set\n",
		}
		for i := range scN {
			files[fmt.Sprintf("dumps/b%02d.json", i)] = scStatSources(i)
		}
		for rel, b := range extra {
			files[rel] = b
		}
		for rel, b := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(b), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func scSpec(bin string, args ...string) difftest.Spec {
	common := []string{
		"--our-dump", "ours.txt.gz",
		"--blob", "blob/output.jsonl.gz",
		"--dumps", "dumps",
		"--min-support", "2",
	}
	full := append(append([]string{}, common...), args...)
	return difftest.Spec{
		Tool:         "source-census",
		PythonScript: scPy,
		PythonArgs:   full,
		GoBin:        bin,
		GoArgs:       append([]string{"source-census"}, full...),
		Stderr:       difftest.StderrExact,
		RefusalSites: 7,
	}
}

// TestSourceCensusPortIsIndistinguishable: THE OPTION SWEEP IS THE POINT: every flag changes a BRANCH, not a format.
func TestSourceCensusPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := scSandbox(t, repo, nil)

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"one-stat", []string{"--stat", "Life"}},
		{"two-stats", []string{"--stat", "Life", "--stat", "Armour"}},
		{"self-check", []string{"--stat", "Life", "--self-check"}},
		{"direction-high", []string{"--stat", "Life", "--direction", "high"}},
		{"direction-low", []string{"--stat", "Life", "--direction", "low"}},
		{"granular", []string{"--stat", "Life", "--granular"}},
		{"fail-class-miss", []string{"--stat", "Life", "--fail-class", "miss"}},
		{"fail-class-extra", []string{"--stat", "Life", "--fail-class", "extra"}},
		{"top-truncates", []string{"--stat", "Life", "--top", "2"}},
		{"support-excludes-everything", []string{"--stat", "Life", "--min-support", "999"}},
		{"all-agree", []string{"--stat", "Armour"}},
		{"tiny-values-pin-the-absolute-epsilon", []string{"--stat", "Tiny"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), scSpec(bin, tc.args...),
				difftest.Fixture{Name: tc.name, Passes: true, Break: func(string) error { return nil }}, mk)
		})
	}
}

func TestSourceCensusRefusals(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := scSandbox(t, repo, nil)

	pySrc := difftest.ReadOriginal(t, repo, scPy)

	spec := scSpec(bin, "--stat", "Life")
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "run-dump-missing", Refusal: "ours.txt.gz is not readable:",
			Break: func(sb string) error { return os.Remove(filepath.Join(sb, "ours.txt.gz")) }},

		{Name: "run-dump-not-gzip", Refusal: "ours.txt.gz is not a gzip file",
			Break: func(sb string) error { return put(sb, "ours.txt.gz", "plain text, not gzip") }},

		{Name: "run-dump-has-no-builds", Refusal: "contains no build records",
			Break: func(sb string) error {
				return put(sb, "ours.txt.gz", scGz(t, "V\tdataVersion\t3.29\n"))
			}},

		{Name: "stat-before-any-build", Refusal: "has an S record before any B record",
			Break: func(sb string) error {
				return put(sb, "ours.txt.gz", scGz(t, "S\tLife\t1.0\nB\tb00.xml\tok\n"))
			}},

		{Name: "stat-value-hex-float", Refusal: "has a non-numeric value '0x1p-2'",
			Break: func(sb string) error {
				return put(sb, "ours.txt.gz", scGz(t, "B\tb00.xml\tok\nS\tLife\t0x1p-2\n"))
			}},

		{Name: "stat-value-not-numeric", Refusal: "has a non-numeric value",
			Break: func(sb string) error {
				return put(sb, "ours.txt.gz", scGz(t, "B\tb00.xml\tok\nS\tLife\tnot-a-number\n"))
			}},

		{Name: "blob-line-not-json", Refusal: "line 2 is not valid JSON",
			Break: func(sb string) error {
				return put(sb, "blob/output.jsonl.gz", scGz(t, `{"build":"b00.xml"}`+"\n{ nope\n"))
			}},

		{Name: "blob-record-has-no-build", Refusal: "line 1 has no 'build' field",
			Break: func(sb string) error {
				return put(sb, "blob/output.jsonl.gz", scGz(t, `{"output":{}}`+"\n"))
			}},

		{Name: "blob-build-not-a-string", Refusal: "line 1 has a non-string 'build'",
			Break: func(sb string) error {
				return put(sb, "blob/output.jsonl.gz", scGz(t, `{"build":5,"output":{}}`+"\n"))
			}},

		{Name: "dumps-is-not-a-directory", Refusal: "dumps is not a directory",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "dumps")) }},

		{Name: "dump-set-covers-too-few", Refusal: "covers only",
			Break: func(sb string) error {
				for i := range 6 {
					if err := os.Remove(filepath.Join(sb, fmt.Sprintf("dumps/b%02d.json", i))); err != nil {
						return fmt.Errorf("fixture: %w", err)
					}
				}
				return nil
			}},

		{Name: "no-build-is-graded", Refusal: "no build is present in BOTH the run dump and the oracle blob",
			Break: func(sb string) error {
				var b []byte
				for i := range scN {
					j, _ := json.Marshal(map[string]any{
						"build": fmt.Sprintf("b%02d.xml", i), "treeSources": 1,
						"output": map[string]any{"Life": scPoBLife(i)},
					})
					b = append(append(b, j...), '\n')
				}
				return put(sb, "blob/output.jsonl.gz", scGz(t, string(b)))
			}},

		{Name: "tree-sources-not-a-number", Refusal: "treeSources is 'many', which is not a number",
			Break: func(sb string) error {
				return put(sb, "blob/output.jsonl.gz",
					scGz(t, `{"build":"b00.xml","treeSources":"many","output":{}}`+"\n"))
			}},

		{Name: "statsources-entry-not-a-table", Refusal: "entry is not a table",
			Break: func(sb string) error {
				return put(sb, "dumps/b06.json",
					`{"statSources":{"Life":{"MORE":["a string, not a table"]}}}`)
			}},

		{Name: "pob-dump-not-json", Refusal: "b06.json is not valid JSON",
			Break: func(sb string) error { return put(sb, "dumps/b06.json", "{ not json") }},

		{Name: "pob-dump-not-an-object", Refusal: "b06.json is not a JSON object",
			Break: func(sb string) error { return put(sb, "dumps/b06.json", `["statSources"]`) }},

		{Name: "pob-dump-unreadable", Refusal: "b06.json is not readable:",
			Break: func(sb string) error {
				return os.Chmod(filepath.Join(sb, "dumps/b06.json"), 0o000)
			}},

		{Name: "statsources-not-a-table", Refusal: "'statSources' is not a table",
			Break: func(sb string) error {
				return put(sb, "dumps/b06.json", `{"statSources": "nope"}`)
			}},

		{Name: "one-stats-sources-not-a-table", Refusal: "statSources['Life'] is not a table",
			Break: func(sb string) error {
				return put(sb, "dumps/b06.json", `{"statSources": {"Life": []}}`)
			}},

		{Name: "source-is-not-a-string", Refusal: "entry has a non-string 'source'",
			Break: func(sb string) error {
				return put(sb, "dumps/b06.json", `{"statSources":{"Life":{"MORE":[{"source":7}]}}}`)
			}},

		{Name: "blob-line-not-an-object", Refusal: "line 1 is not a JSON object",
			Break: func(sb string) error {
				return put(sb, "blob/output.jsonl.gz", scGz(t, "[1,2,3]\n"))
			}},

		{Name: "blob-not-gzip", Refusal: "output.jsonl.gz is not a gzip file",
			Break: func(sb string) error { return put(sb, "blob/output.jsonl.gz", "plain text") }},

		{Name: "run-dump-not-text", Refusal: "ours.txt.gz is truncated, or is not text",
			Break: func(sb string) error {
				return put(sb, "ours.txt.gz", scGz(t, "B\tb00.xml\tok\nS\tLife\t\xff\xfe1.0\n"))
			}},

		{Name: "blob-output-not-a-table", Refusal: "'output' is 'nope', which is not a table",
			Break: func(sb string) error {
				return put(sb, "blob/output.jsonl.gz",
					scGz(t, `{"build":"b00.xml","treeSources":9,"output":"nope"}`+"\n"))
			}},

		{Name: "graded-value-not-a-number", Refusal: "'Life' is 'lots', which is not a number",
			Break: func(sb string) error {
				var b []byte
				for i := range scN {
					out := map[string]any{"Life": scPoBLife(i)}
					if i == 7 {
						out["Life"] = "lots"
					}
					j, _ := json.Marshal(map[string]any{
						"build": fmt.Sprintf("b%02d.xml", i), "treeSources": 9, "output": out,
					})
					b = append(append(b, j...), '\n')
				}
				return put(sb, "blob/output.jsonl.gz", scGz(t, string(b)))
			}},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if os.Geteuid() == 0 && f.Name == "pob-dump-unreadable" {
				t.Skip("running as root, so a mode-000 file is still readable")
			}
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}

	difftest.AssertKeyDeletionAgrees(t, t.Context(), spec, []string{"dumps/b06.json"}, mk)
}

// TestSourceCensusRefusesAStatOutsideTheOutputTable: THE WRONG-TABLE REFUSAL IS THE ONE THIS TOOL EXISTS TO AVOID GETTING WRONG.
func TestSourceCensusRefusesAStatOutsideTheOutputTable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := scSandbox(t, repo, nil)

	for _, tc := range []struct{ name, stat, refusal string }{
		{"present-in-another-table", "MinionLife", "It IS present in: minionOutput"},
		{"present-nowhere", "Nonesuch", "does not appear in the blob's `output` table"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(), scSpec(bin, "--stat", tc.stat),
				difftest.Fixture{Name: tc.name, Refusal: tc.refusal,
					Break: func(string) error { return nil }}, mk)
		})
	}
}

// TestSourceCensusArgumentErrors: ARGUMENT ERRORS NEED THEIR OWN SPECS BECAUSE ARGS LIVE ON THE SPEC.
func TestSourceCensusArgumentErrors(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)

	for _, tc := range []struct {
		name    string
		args    []string
		refusal string
	}{
		{"unknown-argument", []string{"--stat", "Life", "--nope"}, "unknown argument '--nope'"},
		{"value-missing", []string{"--stat"}, "--stat needs a value"},
		{"bad-choice", []string{"--stat", "Life", "--direction", "sideways"},
			"--direction must be one of both, high, low, not 'sideways'"},
		{"bad-integer", []string{"--stat", "Life", "--top", "lots"},
			"--top needs an integer, not 'lots'"},
		{"integer-with-whitespace-is-fine", []string{"--stat", "Life", "--top", " 3 "}, ""},
		{"float-is-not-an-integer", []string{"--stat", "Life", "--top", "3.0"},
			"--top needs an integer, not '3.0'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mk := scSandbox(t, repo, nil)
			f := difftest.Fixture{Name: tc.name, Refusal: tc.refusal,
				Break: func(string) error { return nil }}
			if tc.refusal == "" {
				f = difftest.Fixture{Name: tc.name, Passes: true,
					Break: func(string) error { return nil }}
			}
			difftest.RunFixture(t, t.Context(), scSpec(bin, tc.args...), f, mk)
		})
	}

	for _, tc := range []struct {
		name    string
		args    []string
		refusal string
	}{
		{"our-dump-required", []string{"--stat", "Life"}, "--our-dump is required"},
		{"stat-required", []string{"--our-dump", "ours.txt.gz"}, "at least one --stat is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mk := scSandbox(t, repo, nil)
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool: "source-census", PythonScript: scPy, PythonArgs: tc.args,
				GoBin: bin, GoArgs: append([]string{"source-census"}, tc.args...),
				Stderr: difftest.StderrExact,
			}, difftest.Fixture{Name: tc.name, Refusal: tc.refusal,
				Break: func(string) error { return nil }}, mk)
		})
	}
}

// TestSourceCensusScansOnlyTheFirst200CoveredBuilds: THE "It IS present in:" SCAN READS ONLY THE FIRST 200 COVERED BUILDS.
func TestSourceCensusScansOnlyTheFirst200CoveredBuilds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)

	const n = 260
	dump := make([]byte, 0, n*40)
	var blob []byte
	extra := map[string]string{}
	for i := range n {
		build := fmt.Sprintf("c%03d.xml", i)
		dump = append(dump, fmt.Sprintf("B\t%s\tok\nS\tLife\t%d\n", build, 100+i)...)
		rec := map[string]any{
			"build": build, "treeSources": 9,
			"output": map[string]any{"Life": float64(100 + i)},
		}
		if i == 250 {
			rec["lateOutput"] = map[string]any{"Hidden": 1.0}
		}
		j, _ := json.Marshal(rec)
		blob = append(append(blob, j...), '\n')
		extra[fmt.Sprintf("dumps/c%03d.json", i)] = `{"statSources":{}}`
	}
	extra["ours.txt.gz"] = scGz(t, "V\tdataVersion\t3.29\n"+string(dump))
	extra["blob/output.jsonl.gz"] = scGz(t, string(blob))
	for i := range scN {
		extra[fmt.Sprintf("dumps/b%02d.json", i)] = ""
	}
	mk := scSandbox(t, repo, extra)

	difftest.RunFixture(t, t.Context(), scSpec(bin, "--stat", "Hidden"),
		difftest.Fixture{Name: "hidden-past-the-cutoff",
			Refusal: "does not appear in the blob's `output` table on ANY of 260 graded builds",
			Break:   func(string) error { return nil }}, mk)
}

// TestSourceCensusSpellsNanAndInfLikePython: THIS FIXTURE FOUND A REAL BUG IN THE ORIGINAL, not a porting slip.
func TestSourceCensusSpellsNanAndInfLikePython(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)

	var blob []byte
	for i := range scN {
		life := scPoBLife(i)
		if i >= 6 {
			life = 0
		}
		j, _ := json.Marshal(map[string]any{
			"build": fmt.Sprintf("b%02d.xml", i), "treeSources": 9,
			"output": map[string]any{"Life": life},
		})
		blob = append(append(blob, j...), '\n')
	}
	extra := map[string]string{"blob/output.jsonl.gz": scGz(t, string(blob))}
	for _, i := range []int{0, 1, 2, 3, 4, 5, 12} {
		extra[fmt.Sprintf("dumps/b%02d.json", i)] = `{"statSources":{"Life":{"value":0}}}`
	}
	mk := scSandbox(t, repo, extra)

	difftest.RunFixture(t, t.Context(), scSpec(bin, "--stat", "Life"),
		difftest.Fixture{Name: "nan-median-and-inf-baseline", Passes: true,
			Break: func(string) error { return nil }}, mk)
}

func TestSourceCensusIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := scSandbox(t, repo, nil)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "source-census",
		PythonScript: scPy,
		PythonArgs: []string{
			"--our-dump", "ours.txt.gz", "--blob", "blob/output.jsonl.gz",
			"--dumps", "dumps", "--min-support", "2", "--stat", "Life",
		},
	}, mk, []string{"0", "12345", "999983"})
}
