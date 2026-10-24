package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
)

const pdtsPy = "oracle/pob_data_tables_stamp.py"

func pdtsRef() string {
	b, _ := json.Marshal(map[string]any{
		"tables": map[string]any{
			"TotemLife":      []any{1.0, 1.05, 1.1},
			"MaximumBrittle": 0.5,
			"StoredUses":     3,
			"EmptyTable":     map[string]any{},
			"EmptyList":      []any{},
			"Nested":         map[string]any{"a": []any{map[string]any{"deep": true}}},
		},
		"source": "oracle/pob_data_tables.lua",
	})
	return string(b)
}

func pdtsSandbox(t *testing.T, repo string, extra map[string]string) func() (string, error) {
	t.Helper()
	stamp := difftest.ReadOriginal(t, repo, pdtsPy)
	pin := difftest.ReadOriginal(t, repo, "scripts/pob_pin.py")
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		files := map[string]string{
			pdtsPy:                        string(stamp),
			"pob/src/Modules/Main.lua":    "-- Main\n",
			"pob/src/Launch.lua":          "-- Launch\n",
			"pob/src/HeadlessWrapper.lua": "-- Headless\n",
			"pob/runtime/lua/dkjson.lua":  "-- dkjson\n",
			"refs/tables.json":            pdtsRef(),
			"scripts/.keep":               "",
		}
		if pin != nil {
			files["scripts/pob_pin.py"] = string(pin)
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
		const when = "2000-01-01T00:00:00 +0000"
		for _, args := range [][]string{
			{"init", "-q"}, {"add", "-A"},
			{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-qm", "pin"},
		} {
			cmd := exec.CommandContext(t.Context(), "git", args...) //nolint:gosec // the harness's own sandbox path
			cmd.Dir = filepath.Join(sb, "pob")
			cmd.Env = append(os.Environ(),
				"GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("sandbox: git %s: %w: %s", args[0], err, out)
			}
		}
		rec := exec.CommandContext(t.Context(), buildPort(t, repo), "pob-pin", "--record") //nolint:gosec // the harness's own binary
		rec.Dir = sb
		rec.Env = append(os.Environ(), "POB_DIR="+filepath.Join(sb, "pob"))
		if out, err := rec.CombinedOutput(); err != nil {
			return "", fmt.Errorf("sandbox: --record: %w: %s", err, out)
		}
		return sb, nil
	}
}

func pdtsSpec(bin string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "pob-data-tables-stamp",
		PythonScript: pdtsPy,
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"pob-data-tables-stamp"}, args...),
		Stderr:       difftest.StderrExact,
		Env:          []string{"POB_DIR=pob"},
		RefusalSites: 2,
	}
}

// TestPobDataTablesStampWritesIdenticalStamps: THE SUCCESS PATH IS ITS OWN TEST BECAUSE IT IS THE ONLY ONE THAT WRITES ANYTHING.
func TestPobDataTablesStampWritesIdenticalStamps(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pdtsSandbox(t, repo, nil)

	spec := pdtsSpec(bin, "refs")
	spec.Files = []string{"refs/tables.json"}
	difftest.RunFixture(t, t.Context(), spec,
		difftest.Fixture{Name: "stamps-the-one-unstamped-reference", Passes: true,
			Break: func(string) error { return nil }}, mk)

	direct := pdtsSpec(bin, "refs/tables.json")
	direct.Files = []string{"refs/tables.json"}
	difftest.RunFixture(t, t.Context(), direct,
		difftest.Fixture{Name: "stamps-a-named-reference", Passes: true,
			Break: func(string) error { return nil }}, mk)

	sweep := pdtsSpec(bin, "refs")
	difftest.AssertKeyDeletionAgrees(t, t.Context(), sweep, []string{"refs/tables.json"}, mk)
}

func TestPobDataTablesStampPortIsIndistinguishable(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pdtsSandbox(t, repo, nil)

	pySrc := difftest.ReadOriginal(t, repo, pdtsPy)

	spec := pdtsSpec(bin, "refs")
	spec.Fixtures = []difftest.Fixture{
		{Name: "happy", Passes: true},

		{Name: "reference-unreadable", Refusal: "refs/tables.json is not readable:",
			Break: func(sb string) error {
				return os.Chmod(filepath.Join(sb, "refs/tables.json"), 0o000)
			}},

		{Name: "reference-invalid-json", Refusal: "refs/tables.json is not valid JSON",
			Break: func(sb string) error { return put(sb, "refs/tables.json", "{ not json") }},

		{Name: "reference-not-an-object", Refusal: "refs/tables.json is not a JSON object",
			Break: func(sb string) error { return put(sb, "refs/tables.json", `["tables"]`) }},

		{Name: "reference-declares-no-tables", Refusal: "declares no tables",
			Break: func(sb string) error { return put(sb, "refs/tables.json", `{"tables": {}}`) }},

		{Name: "reference-unwritable", Refusal: "cannot write refs/tables.json:",
			Break: func(sb string) error {
				return os.Chmod(filepath.Join(sb, "refs/tables.json"), 0o400)
			}},

		{Name: "two-unstamped-references", Refusal: "2 unstamped references in refs:",
			Break: func(sb string) error { return put(sb, "refs/second.json", pdtsRef()) }},

		{Name: "no-pob-checkout", Refusal: "cannot establish the pin identity:",
			Break: func(sb string) error { return os.RemoveAll(filepath.Join(sb, "pob")) }},

		{Name: "no-pin-record", Refusal: "cannot establish the pin identity:",
			Break: func(sb string) error { return os.Remove(filepath.Join(sb, "scripts/pob_pin.json")) }},

		{Name: "checkout-is-not-the-pin", Refusal: "is NOT the pinned PoB",
			Break: func(sb string) error { return put(sb, "pob/src/Launch.lua", "-- edited\n") }},
	}

	difftest.AssertRefusalSiteCount(t, spec, pySrc)

	for _, f := range spec.Fixtures {
		t.Run(f.Name, func(t *testing.T) {
			if os.Geteuid() == 0 &&
				(f.Name == "reference-unreadable" || f.Name == "reference-unwritable") {
				t.Skip("running as root, so file modes do not restrict anything")
			}
			difftest.RunFixture(t, t.Context(), spec, f, mk)
		})
	}
}

// TestPobDataTablesStampUsage: THE ARGUMENT-COUNT PATH NEEDS ITS OWN SPEC BECAUSE ARGS LIVE ON THE SPEC, NOT THE FIXTURE.
func TestPobDataTablesStampUsage(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := pdtsSandbox(t, repo, nil)

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no-arguments", nil},
		{"two-arguments", []string{"refs", "extra"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := pdtsSpec(bin, tc.args...)
			difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{
				Name:    tc.name,
				Refusal: "usage: pob_data_tables_stamp.py <reference.json|reference-dir>",
				Break:   func(string) error { return nil },
			}, mk)
		})
	}
}

// TestPobDataTablesStampNormalisesItsArgumentLikePathlib: THE ARGUMENT IS NORMALISED BY pathlib BEFORE IT IS EVER PRINTED, AND NOTHING ELSE HERE CAN SEE THAT.
func TestPobDataTablesStampNormalisesItsArgumentLikePathlib(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)

	for _, tc := range []struct{ spelling, printed string }{
		{"refs/", "refs"},
		{"./refs", "refs"},
		{"refs//", "refs"},
		{"./refs/", "refs"},
		{"refs/../refs", "refs/../refs"},
	} {
		t.Run(tc.spelling, func(t *testing.T) {
			mk := pdtsSandbox(t, repo, nil)
			spec := pdtsSpec(bin, tc.spelling)
			difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{
				Name:    "two-unstamped",
				Refusal: "2 unstamped references in " + tc.printed + ": second.json, tables.json.",
				Break:   func(sb string) error { return put(sb, "refs/second.json", pdtsRef()) },
			}, mk)
		})
	}
}

// TestPobDataTablesStampIsIdempotent: EXIT 0 WITH A MESSAGE IS A THIRD OUTCOME AND IT IS THE RE-RUN CASE.
func TestPobDataTablesStampIsIdempotent(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	stamped, _ := json.Marshal(map[string]any{
		"tables": map[string]any{"TotemLife": []any{1.0}},
		"pobPin": map[string]any{"content_sha256": "deadbeef", "file_count": 4},
	})
	mk := pdtsSandbox(t, repo, map[string]string{"refs/tables.json": string(stamped)})

	spec := pdtsSpec(bin, "refs")
	spec.Files = []string{"refs/tables.json"}
	difftest.RunFixture(t, t.Context(), spec, difftest.Fixture{
		Name: "nothing-to-stamp", Passes: true, Break: func(string) error { return nil },
	}, mk)
}

func TestPobDataTablesStampIsStableAcrossHashSeeds(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	mk := pdtsSandbox(t, repo, nil)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "pob-data-tables-stamp",
		PythonScript: pdtsPy,
		PythonArgs:   []string{"refs"},
		Env:          []string{"POB_DIR=pob"},
	}, mk, []string{"0", "12345", "999983"})
}
