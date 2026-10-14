package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/difftest"
	"github.com/Vilsol/go-pob/scripts/gopob/internal/toolchain"
)

const nrSh = "scripts/nice.sh"

const nrProbe = `#!/bin/sh
echo "niceness: $(nice)"
ionice -p $$ 2>/dev/null || echo "ionice: unavailable"
echo "GOPOB_NICED=${GOPOB_NICED:-unset}"
echo "args: $*"
echo "to stderr" >&2
exit "${PROBE_EXIT:-0}"
`

func nrSandbox(t *testing.T, repo string) func() (string, error) {
	t.Helper()
	sh := difftest.ReadOriginal(t, repo, nrSh)
	base := t.TempDir()
	n := 0
	return func() (string, error) {
		n++
		sb := filepath.Join(base, fmt.Sprintf("sb%d", n))
		if err := os.MkdirAll(filepath.Join(sb, "scripts"), 0o750); err != nil {
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.WriteFile(filepath.Join(sb, nrSh), sh, 0o700); err != nil { //nolint:gosec
			return "", fmt.Errorf("sandbox: %w", err)
		}
		if err := os.WriteFile(filepath.Join(sb, "probe.sh"), []byte(nrProbe), 0o700); err != nil { //nolint:gosec
			return "", fmt.Errorf("sandbox: %w", err)
		}
		return sb, nil
	}
}

func nrSpec(bin string, env []string, unset []string, args ...string) difftest.Spec {
	return difftest.Spec{
		Tool:         "nice",
		PythonScript: nrSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   args,
		GoBin:        bin,
		GoArgs:       append([]string{"nice"}, args...),
		Stderr:       difftest.StderrExact,
		Env:          env,
		EnvUnset:     unset,
		RefusalSites: 2,
	}
}

// TestNiceRunPortIsIndistinguishable: GOPOB_NICED MUST BE **UNSET**, NOT MERELY EMPTY, OR THIS TEST IS VACUOUS.
func TestNiceRunPortIsIndistinguishable(t *testing.T) {
	requirePOSIX(t)
	nrPinBaseNiceness(t)
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	mk := nrSandbox(t, repo)

	shSrc := difftest.ReadOriginal(t, repo, nrSh)

	type fixture struct {
		difftest.Fixture
		env   []string
		unset []string
		args  []string
	}
	probe := []string{"./probe.sh", "one", "two"}
	fixtures := []fixture{
		{unset: []string{"GOPOB_NICED"}, args: probe,
			Fixture: difftest.Fixture{Name: "happy"}},

		{unset: []string{"GOPOB_NICED"}, args: probe,
			env: []string{"NICE=3"},
			Fixture: difftest.Fixture{Name: "nice-level-override", Passes: true,
				Break: func(string) error { return nil }}},

		{unset: []string{"GOPOB_NICED"}, args: probe,
			env: []string{"NICE=0"},
			Fixture: difftest.Fixture{Name: "nice-zero-disables", Passes: true,
				Break: func(string) error { return nil }}},

		{env: []string{"GOPOB_NICED=1"}, args: probe,
			Fixture: difftest.Fixture{Name: "already-niced-passes-through", Passes: true,
				Break: func(string) error { return nil }}},

		{unset: []string{"GOPOB_NICED"}, args: probe,
			Fixture: difftest.Fixture{Name: "exit-status-is-the-command's", Finding: true,
				Break: func(sb string) error {
					return put(sb, "probe.sh", "#!/bin/sh\necho 'probe: failing on purpose'\nexit 7\n")
				}}},

		{unset: []string{"GOPOB_NICED"}, args: nil,
			Fixture: difftest.Fixture{Name: "no-arguments",
				Refusal: "usage: scripts/nice.sh <command> [args...]",
				Break:   func(string) error { return nil }}},

		{unset: []string{"GOPOB_NICED"}, args: []string{"definitely-not-a-real-command"},
			Fixture: difftest.Fixture{Name: "command-not-found",
				Refusal: "scripts/nice.sh: definitely-not-a-real-command: command not found",
				Break:   func(string) error { return nil }}},
	}

	table := make([]difftest.Fixture, 0, len(fixtures))
	for _, f := range fixtures {
		table = append(table, f.Fixture)
	}
	spec := nrSpec(bin, nil, []string{"GOPOB_NICED"}, probe...)
	spec.Fixtures = table
	difftest.AssertRefusalSiteCount(t, spec, shSrc)

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			difftest.RunFixture(t, t.Context(),
				nrSpec(bin, f.env, f.unset, f.args...), f.Fixture, mk)
		})
	}
}

const nrGoldenBaseNiceness = 10

func nrPinBaseNiceness(t *testing.T) {
	t.Helper()
	raw, err := rawGetpriority()
	if err != nil {
		t.Fatalf("getpriority: %v", err)
	}
	cur := 20 - raw
	if cur > nrGoldenBaseNiceness {
		t.Skipf("running at niceness %d; the goldens were recorded at %d and niceness cannot be lowered", cur, nrGoldenBaseNiceness)
	}
	toolchain.SetprioritySelf(nrGoldenBaseNiceness)
	toolchain.IoprioIdleSelf()
}

func TestNiceRunIsStableAcrossLocales(t *testing.T) {
	requirePOSIX(t)
	repo := repoRoot(t)
	mk := nrSandbox(t, repo)
	difftest.SelfIdentity(t, t.Context(), difftest.Spec{
		Tool:         "nice",
		PythonScript: nrSh,
		Interpreter:  "bash",
		SeedVar:      "LC_ALL",
		PythonArgs:   []string{"./probe.sh"},
		EnvUnset:     []string{"GOPOB_NICED"},
	}, mk, []string{"C", "en_US.utf8", "C.utf8"})
}

// TestNiceRunTargetNicenessDecodesGetpriority: THIS TABLE EXISTS BECAUSE THE DIFFERENTIAL IS BLIND HERE, NOT AS BELT AND BRACES.
func TestNiceRunTargetNicenessDecodesGetpriority(t *testing.T) {
	for _, tc := range []struct{ raw, inc, want int }{
		{20, 10, 10},
		{10, 10, 20},
		{10, 3, 13},
		{15, 10, 15},
		{0, 5, 25},
		{25, 10, 5},
	} {
		if got := nrTargetNiceness(tc.raw, tc.inc); got != tc.want {
			t.Errorf("nrTargetNiceness(raw=%d, inc=%d) = %d, want %d (caller niceness %d)",
				tc.raw, tc.inc, got, tc.want, 20-tc.raw)
		}
	}
}
