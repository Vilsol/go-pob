package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const mbPy = "oracle/modal_blob.py"

var mbRuns = 4

func mbRecord(build string, tree string, err string, tables map[string]map[string]string) string {
	var parts []string
	parts = append(parts, `"build": `+pyJSONString(build))
	if tree != "" {
		parts = append(parts, `"treeSources": `+tree)
	}
	if err != "" {
		parts = append(parts, `"error": `+pyJSONString(err))
	}
	for _, table := range mbTables {
		vals, ok := tables[table]
		if !ok {
			continue
		}
		var kv []string
		for _, k := range mbSortedKeys(vals) {
			if vals[k] == "" {
				continue
			}
			kv = append(kv, pyJSONString(k)+": "+vals[k])
		}
		parts = append(parts, pyJSONString(table)+": {"+strings.Join(kv, ", ")+"}")
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func mbSortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func mbRun(r int) string {
	pick := func(vs ...string) string { return vs[r] }
	main := map[string]string{
		"Unanimous":   "1.5",
		"Cluster":     pick("1.0", "1.0000000001", "1.0", "1.0"),
		"FarApart":    pick("1.0", "1.0", "1.0", "1.001"),
		"Split22":     pick("1.0", "1.0", "2.0", "2.0"),
		"AbsentWins":  pick("7.5", "", "", ""),
		"AbsentLoses": pick("7.5", "7.5", "7.5", ""),
		"TieSmallest": pick("1.0", "1.0000000001", "1.0", "1.0000000001"),
		"Dropped2":    pick("3.0", "3.0", "4.0", "4.0"),
		"Dropped3":    pick("5.0", "5.0", "6.0", "6.0"),
		"NearZero":    pick("0.0", "1e-13", "0.0", "0.0"),
		"AbsentTie":   pick("4.0", "4.0", "", ""),
	}
	for i := range 15 {
		main[fmt.Sprintf("Tied%02d", i)] = pick("7.0", "7.0", "8.0", "8.0")
	}
	main["ZZShared"] = pick("1.0", "1.0", "2.0", "2.0")
	a := mbRecord("0001-alpha.xml", "46", "", map[string]map[string]string{"output": main})

	bTables := map[string]map[string]string{
		"output":         {"Plain": "2.0", "ZZShared": pick("1.0", "1.0", "2.0", "2.0")},
		"outputMainHand": {"MH": pick("1.0", "1.0", "1.0", "1.0")},
	}
	if r == 3 {
		delete(bTables, "outputMainHand")
	}
	b := mbRecord("0002-beta.xml", pick("46", "44", "46", "44"), "", bTables)

	cTables := map[string]map[string]string{
		"output": {"Only": "9.0", "ZZShared": pick("1.0", "1.0", "2.0", "2.0")},
	}
	if r == 0 {
		cTables["enemyOutput"] = map[string]string{"E": "1.0"}
	}
	c := mbRecord("0003-gamma.xml", "50", "", cTables)

	d := mbRecord("0004-delta.xml", "0", "headless PoB blew up — ünïcode", nil)

	e := mbRecord("0005-epsilon.xml", "47.5", "", map[string]map[string]string{
		"output": {"Plain": "1.0"},
	})

	return a + "\n" + b + "\n" + c + "\n" + d + "\n" + e + "\n"
}

func mbSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	py := difftest.ReadOriginal(t, repo, mbPy)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{mbPy: string(py)}
		for r := range mbRuns {
			files[fmt.Sprintf("r%d.jsonl", r)] = mbRun(r)
		}
		files["short.jsonl"] = strings.SplitAfter(mbRun(0), "\n")[0]
		files["desync.jsonl"] = strings.Replace(mbRun(0), "0002-beta.xml", "0002-OTHER.xml", 1)
		files["notjson.jsonl"] = strings.Replace(mbRun(0), "{", "not json ", 1)
		files["nobuild.jsonl"] = strings.Replace(mbRun(0), `"build": "0001-alpha.xml", `, "", 1)
		files["outdir/keep"] = "the output path is a directory\n"
		for rel, body := range files {
			dst := filepath.Join(sb, rel)
			if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}
}

func mbArgs(extra ...string) []string {
	a := []string{"out.jsonl", "report.txt"}
	for r := range mbRuns {
		a = append(a, fmt.Sprintf("r%d.jsonl", r))
	}
	return append(a, extra...)
}

func TestModalBlobPortIsIndistinguishable(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := mbSandbox(t, repo)

	pySrc := difftest.ReadOriginal(t, repo, mbPy)

	base := mbArgs()
	spec := difftest.Spec{
		Tool:         "modal-blob",
		PythonScript: mbPy,
		PythonArgs:   base,
		GoBin:        bin,
		GoArgs:       append([]string{"modal-blob"}, base...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.jsonl", "report.txt"},
		RefusalSites: 3,
		Fixtures: []difftest.Fixture{
			{Name: "happy"},

			{Name: "no-args", Refusal: "usage: modal_blob.py <out.jsonl> <report.txt>",
				Break: func(string) error { return nil }},
			{Name: "too-few-runs-unreachable", NoFixture: true,
				Refusal: "modal: modal_blob: need at least 3 runs to have a mode",
				Note: "the usage guard requires 5 positional arguments, so n >= 3 always; " +
					"covered by the no-args fixture, which is what actually fires"},
			{Name: "four-positional-args", Refusal: "usage: modal_blob.py <out.jsonl> <report.txt>",
				Break: func(string) error { return nil }},
			{Name: "min-fraction-not-a-number", Refusal: "modal: --min-fraction must be a number, not 'abc'",
				Break: func(string) error { return nil }},
			{Name: "input-missing", Refusal: "modal: nope.jsonl is not readable: No such file or directory",
				Break: func(string) error { return nil }},
			{Name: "record-counts-differ", Refusal: "modal: runs have different record COUNTS",
				Break: func(string) error { return nil }},
			{Name: "desynchronised", Refusal: "modal: runs desynchronised at ['0002-OTHER.xml', '0002-beta.xml']",
				Break: func(string) error { return nil }},
			{Name: "line-not-json", Refusal: "modal: notjson.jsonl: line 1 is not valid JSON",
				Break: func(string) error { return nil }},
			{Name: "line-has-no-build", Refusal: "modal: nobuild.jsonl: line 1 has no build name",
				Break: func(string) error { return nil }},
			{Name: "out-path-is-a-directory", Refusal: "modal: cannot write outdir: Is a directory",
				Break: func(string) error { return nil }},
			{Name: "report-path-is-a-directory", Refusal: "modal: cannot write outdir: Is a directory",
				Break: func(string) error { return nil }},

			{Name: "treesources-is-a-string-in-every-run",
				Refusal: "modal: 0001-alpha.xml: treeSources is not a number",
				Break:   func(sb string) error { return mbPutTree(sb, []string{`"nope"`, `"nope"`, `"nope"`, `"nope"`}) }},
			{Name: "treesources-is-a-string-in-one-run",
				Refusal: "modal: 0001-alpha.xml: treeSources is not a number",
				Break:   func(sb string) error { return mbPutTree(sb, []string{`"nope"`, "3", "3", "3"}) }},
			{Name: "treesources-is-a-boolean",
				Refusal: "modal: 0001-alpha.xml: treeSources is not a number",
				Break:   func(sb string) error { return mbPutTree(sb, []string{"true", "true", "true", "true"}) }},
			{Name: "five-positional-args", Passes: true, Break: func(string) error { return nil }},
		},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		if f.NoFixture {
			continue
		}
		t.Run(f.Name, func(t *testing.T) {
			s := spec
			switch f.Name {
			case "no-args":
				s.PythonArgs = []string{"out.jsonl", "report.txt"}
			case "four-positional-args":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl"}
			case "five-positional-args":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl", "r2.jsonl"}
			case "min-fraction-not-a-number":
				s.PythonArgs = append(mbArgs(), "--min-fraction=abc")
			case "input-missing":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl", "nope.jsonl"}
			case "record-counts-differ":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl", "short.jsonl"}
			case "desynchronised":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl", "desync.jsonl"}
			case "line-not-json":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl", "notjson.jsonl"}
			case "line-has-no-build":
				s.PythonArgs = []string{"out.jsonl", "report.txt", "r0.jsonl", "r1.jsonl", "nobuild.jsonl"}
			case "out-path-is-a-directory":
				s.PythonArgs = append([]string{"outdir", "report.txt"}, mbArgs()[2:]...)
			case "report-path-is-a-directory":
				s.PythonArgs = append([]string{"out.jsonl", "outdir"}, mbArgs()[2:]...)
			}
			if f.Name != "happy" && f.Name != "five-positional-args" {
				s.Files = nil
			}
			s.GoArgs = append([]string{"modal-blob"}, s.PythonArgs...)
			difftest.RunFixture(t, t.Context(), s, f, mk)
		})
	}
}

// TestModalBlobMinFractionSweep: --min-fraction IS A THRESHOLD AND ONE VALUE PINS NOTHING.
func TestModalBlobMinFractionSweep(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := mbSandbox(t, repo)
	for _, f := range []string{"0", "0.25", "0.49", "0.5", "0.51", "0.75", "1"} {
		t.Run("f"+f, func(t *testing.T) {
			args := append(mbArgs(), "--min-fraction="+f)
			difftest.RunFixture(t, t.Context(), difftest.Spec{
				Tool:         "modal-blob",
				PythonScript: mbPy,
				PythonArgs:   args,
				GoBin:        bin,
				GoArgs:       append([]string{"modal-blob"}, args...),
				Stderr:       difftest.StderrExact,
				Files:        []string{"out.jsonl", "report.txt"},
			}, difftest.Fixture{Name: "f" + f}, mk)
		})
	}
}

func TestModalBlobPythonIsStableAcrossHashSeeds(t *testing.T) {
	repo := repoRoot(t)
	mk := mbSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "modal-blob",
		PythonScript: mbPy,
		PythonArgs:   mbArgs(),
		Files:        []string{"out.jsonl", "report.txt"},
	}, mk, []string{"0", "12345", "999983"})
}

// TestModalBlobManyRunsTieOrder: FOURTEEN RUNS, BECAUSE THE CLUSTER SORT CANNOT BE UNSTABLE WITH FOUR.
func TestModalBlobManyRunsTieOrder(t *testing.T) {
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	const runs = 15

	py := difftest.ReadOriginal(t, repo, mbPy)
	base := t.TempDir()
	n := 0
	mk := func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		if err := os.MkdirAll(filepath.Dir(filepath.Join(sb, mbPy)), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.WriteFile(filepath.Join(sb, mbPy), py, 0o600); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		for r := range runs {
			val := 200 + r
			switch r {
			case 13:
				val = 203
			case 14:
				val = 209
			}
			rec := fmt.Sprintf(
				`{"build": "0001-many.xml", "treeSources": %d, "output": {"Spread": %d.5}}`+"\n",
				40+r, val)
			if err := os.WriteFile(filepath.Join(sb, fmt.Sprintf("m%02d.jsonl", r)),
				[]byte(rec), 0o600); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
		}
		return sb, nil
	}

	args := make([]string, 0, runs+3)
	args = append(args, "out.jsonl", "report.txt")
	for r := range runs {
		args = append(args, fmt.Sprintf("m%02d.jsonl", r))
	}
	args = append(args, "--min-fraction=0")
	difftest.RunFixture(t, t.Context(), difftest.Spec{
		Tool:         "modal-blob",
		PythonScript: mbPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"modal-blob"}, args...),
		Stderr:       difftest.StderrExact,
		Files:        []string{"out.jsonl", "report.txt"},
	}, difftest.Fixture{Name: "fourteen-runs"}, mk)
}

func mbPutTree(sb string, perRun []string) error {
	for r, lit := range perRun {
		body := fmt.Sprintf(
			"{\"build\": \"0001-alpha.xml\", \"treeSources\": %s, \"tables\": {}}\n"+
				"{\"build\": \"0002-beta.xml\", \"treeSources\": 3, \"tables\": {}}\n", lit)
		dst := filepath.Join(sb, fmt.Sprintf("r%d.jsonl", r))
		if err := os.WriteFile(dst, []byte(body), 0o600); err != nil {
			return fmt.Errorf("fixture: %w", err)
		}
	}
	return nil
}
