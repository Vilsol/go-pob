package difftest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fake struct {
	failed bool
	msg    string
}

func (f *fake) Helper() {}
func (f *fake) Fatalf(format string, args ...any) {
	f.failed = true
	f.msg = fmt.Sprintf(format, args...)
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
}

func buildPair(t *testing.T, pySrc, goSrc string) (Spec, func() (string, error)) {
	t.Helper()
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "scripts", "tool.py"), pySrc)

	gd := t.TempDir()
	mustWrite(t, filepath.Join(gd, "go.mod"), "module synthetic\n\ngo 1.26.2\n")
	mustWrite(t, filepath.Join(gd, "main.go"), goSrc)
	bin := filepath.Join(t.TempDir(), "port")
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".")
	cmd.Dir = gd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the synthetic port: %v\n%s", err, out)
	}

	root := t.TempDir()
	n := 0
	return Spec{
			Tool:      "synthetic",
			Synthetic: true, PythonScript: "scripts/tool.py",
			GoBin:        bin,
			RefusalSites: RefusalSites([]byte(pySrc)),
		}, func() (string, error) {
			n++
			d := filepath.Join(root, fmt.Sprintf("sb%d", n))
			if err := os.MkdirAll(d, 0o750); err != nil {
				return "", fmt.Errorf("sandbox: %w", err)
			}
			return d, CopyTree(d, src)
		}
}

const pyRefusing = `import sys
if len(sys.argv) > 1 and sys.argv[1] == "bad":
    print("REFUSING: bad", file=sys.stderr)
    sys.exit(3)
print("ok")
`

const goFaithful = `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "bad" {
		fmt.Fprintln(os.Stderr, "REFUSING: bad")
		os.Exit(3)
	}
	fmt.Println("ok")
}
`

const goWrongMessage = `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "bad" {
		fmt.Fprintln(os.Stderr, "REFUSING: something else entirely")
		os.Exit(3)
	}
	fmt.Println("ok")
}
`

func TestAFaithfulPortPasses(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	s.PythonArgs, s.GoArgs = []string{"bad"}, []string{"bad"}
	RunFixture(t, t.Context(), s, Fixture{Name: "refusal"}, mk)
}

// TestStderrIsTheSoleDiscriminator: THE CONTROL FOR "stderr IS NOT OPTIONAL".
func TestStderrIsTheSoleDiscriminator(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goWrongMessage)
	s.PythonArgs, s.GoArgs = []string{"bad"}, []string{"bad"}

	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{Name: "refusal"}, mk)
	if !f.failed {
		t.Fatal("the harness accepted a port whose refusal MESSAGE differs while rc and stdout " +
			"agree; stderr is not being compared")
	}
	if !strings.Contains(f.msg, "stderr differs") {
		t.Fatalf("failed for the wrong reason: %s", f.msg)
	}

	s.Stderr = StderrFirstLine
	f2 := &fake{}
	RunFixture(f2, t.Context(), s, Fixture{Name: "refusal"}, mk)
	if !f2.failed {
		t.Fatal("StderrFirstLine accepted a different first line")
	}
}

// TestStderrFirstLineToleratesATrailingTraceback: StderrFirstLine exists for the 49 uncaught `raise` sites.
func TestStderrFirstLineToleratesATrailingTraceback(t *testing.T) {
	py := `import sys
print("REFUSING: boom", file=sys.stderr)
print("Traceback (most recent call last):", file=sys.stderr)
print("  File \"x\", line 1", file=sys.stderr)
sys.exit(1)
`
	goSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "REFUSING: boom")
	os.Exit(1)
}
`
	s, mk := buildPair(t, py, goSrc)
	s.Stderr = StderrFirstLine
	RunFixture(t, t.Context(), s, Fixture{Name: "traceback"}, mk)

	s.Stderr = StderrExact
	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{Name: "traceback"}, mk)
	if !f.failed {
		t.Fatal("StderrExact accepted a missing traceback")
	}
}

// TestRefusalSitesCountsEveryNonZeroExit: THIS ASSERTED 3 UNTIL 2026-08-19, EXCLUDING `sys.exit(x)` BECAUSE IT IS NOT A LITERAL.
func TestRefusalSitesCountsEveryNonZeroExit(t *testing.T) {
	src := []byte("sys.exit(3)\nsys.exit(0)\nraise SystemExit(2)\nsys.exit( 1 )\nsys.exit(x)\n" +
		"sys.exit()\n")
	if got := RefusalSites(src); got != 4 {
		t.Fatalf("counted %d, want 4: only exit(0) and a bare exit() are not refusals", got)
	}
}

// TestFixtureSetMustCoverEveryRefusalSiteExactlyOnce: WITHOUT THIS, A FIXTURE THAT STOPS REACHING ITS SITE STILL PASSES.
func TestFixtureSetMustCoverEveryRefusalSiteExactlyOnce(t *testing.T) {
	src := []byte("sys.exit(3)\nsys.exit(4)\n")
	base := Spec{Tool: "s", RefusalSites: 2}

	ok := base
	ok.Fixtures = []Fixture{
		{Name: "happy"},
		{Name: "a", Refusal: "one", Break: func(string) error { return nil }},
		{Name: "b", Refusal: "two", Break: func(string) error { return nil }},
	}
	AssertRefusalSiteCount(t, ok, src)

	for name, s := range map[string]Spec{
		"two fixtures claim one site": {Tool: "s", RefusalSites: 2, Fixtures: []Fixture{
			{Name: "happy"},
			{Name: "a", Refusal: "one", Break: func(string) error { return nil }},
			{Name: "b", Refusal: "one", Break: func(string) error { return nil }},
		}},
		"a site has no fixture": {Tool: "s", RefusalSites: 2, Fixtures: []Fixture{
			{Name: "happy"},
			{Name: "a", Refusal: "one", Break: func(string) error { return nil }},
		}},
		"declared count is stale": {Tool: "s", RefusalSites: 9, Fixtures: []Fixture{{Name: "happy"}}},
		"no happy fixture": {Tool: "s", RefusalSites: 2, Fixtures: []Fixture{
			{Name: "a", Refusal: "one", Break: func(string) error { return nil }},
			{Name: "b", Refusal: "two", Break: func(string) error { return nil }},
		}},
	} {
		f := &fake{}
		AssertRefusalSiteCount(f, s, src)
		if !f.failed {
			t.Fatalf("accepted a fixture set where %s", name)
		}
	}
}

func TestSelfIdentityRefusesAShortSweep(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	f := &fake{}
	SelfIdentity(f, t.Context(), s, mk, []string{"0", "1"})
	if !f.failed {
		t.Fatal("accepted a two-seed sweep as a sweep")
	}
}

// TestSelfIdentityReportsVacuousRatherThanStable: 6 OF 11 SCRIPTS ONLY REFUSED UNDER THE SEED SWEEP.
func TestSelfIdentityReportsVacuousRatherThanStable(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	s.PythonArgs = []string{"bad"}
	f := &fake{}
	SelfIdentity(f, t.Context(), s, mk, []string{"0", "12345", "999983"})
	if !f.failed {
		t.Fatal("a sweep in which every run REFUSED was reported as stable")
	}
	if !strings.Contains(f.msg, "VACUOUS") {
		t.Fatalf("failed for the wrong reason: %s", f.msg)
	}
}

func TestSelfIdentityPassesOnAStableScript(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	SelfIdentity(t, t.Context(), s, mk, []string{"0", "12345", "999983"})
}

func TestSelfIdentityCatchesASeedSensitiveScript(t *testing.T) {
	py := `import os, sys
print(hash("go-pob") % 97)
`
	s, mk := buildPair(t, py, goFaithful)
	f := &fake{}
	SelfIdentity(f, t.Context(), s, mk, []string{"0", "12345", "999983"})
	if !f.failed {
		t.Fatal("a script whose output depends on PYTHONHASHSEED passed the sweep")
	}
}

// TestFileContractCatchesADifferentGeneratedFile: THE FILE CONTRACT IS WHAT chunk_init.py NEEDS.
func TestFileContractCatchesADifferentGeneratedFile(t *testing.T) {
	py := `import pathlib
pathlib.Path("out.txt").write_text("alpha\n")
print("wrote")
`
	goSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	_ = os.WriteFile("out.txt", []byte("beta\n"), 0o600)
	fmt.Println("wrote")
}
`
	s, mk := buildPair(t, py, goSrc)

	RunFixture(t, t.Context(), s, Fixture{Name: "generator"}, mk)

	s.Files = []string{"out.txt"}
	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{Name: "generator"}, mk)
	if !f.failed {
		t.Fatal("stdout agreed and the generated files differed; the FILE contract did not fire")
	}
}

// TestImpureSpecIsRefusedRatherThanCompared: SCREEN FOR READ-WHAT-YOU-WRITE BEFORE ASSIGNING THE SIMPLE CONTRACT.
func TestImpureSpecIsRefusedRatherThanCompared(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	s.Impure = "shells `codegraph index`, mutating a DB outside the sandbox"
	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{Name: "happy"}, mk)
	if !f.failed {
		t.Fatal("an impure spec was compared under the simple contract")
	}
}

// TestEachImplementationGetsItsOwnSandbox: EACH IMPLEMENTATION GETS ITS OWN SANDBOX COPY.
func TestEachImplementationGetsItsOwnSandbox(t *testing.T) {
	py := `import pathlib
p = pathlib.Path("marker")
print("second" if p.exists() else "first")
p.write_text("x")
`
	goSrc := `package main

import (
	"fmt"
	"os"
)

func main() {
	if _, err := os.Stat("marker"); err == nil {
		fmt.Println("second")
	} else {
		fmt.Println("first")
	}
	_ = os.WriteFile("marker", []byte("x"), 0o600)
}
`
	s, mk := buildPair(t, py, goSrc)
	RunFixture(t, t.Context(), s, Fixture{Name: "read-what-you-write"}, mk)
}

// TestAFixtureThatReachesNoRefusalIsRejected: THE CONTROL FOR THE GAP THE PACKAGE DOC CLAIMED WAS CLOSED.
func TestAFixtureThatReachesNoRefusalIsRejected(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{
		Name:    "claims-a-refusal-but-breaks-nothing",
		Refusal: "REFUSING: bad",
		Break:   func(string) error { return nil },
	}, mk)
	if !f.failed {
		t.Fatal("the harness accepted a fixture that declares a refusal site and reaches none; " +
			"both sides took the happy path and agreed, so the fixture covers nothing")
	}
	if !strings.Contains(f.msg, "reached no refusal") {
		t.Fatalf("failed for the wrong reason: %s", f.msg)
	}
}

// TestAFixtureThatReachesTheWrongRefusalIsRejected: AND REACHING *A* REFUSAL IS NOT REACHING *THIS* REFUSAL.
func TestAFixtureThatReachesTheWrongRefusalIsRejected(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	s.PythonArgs, s.GoArgs = []string{"bad"}, []string{"bad"}
	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{
		Name:    "reaches-another-site",
		Refusal: "REFUSING: a different site",
		Break:   func(string) error { return nil },
	}, mk)
	if !f.failed {
		t.Fatal("the harness accepted a fixture that reached a refusal other than the one it names")
	}
	if !strings.Contains(f.msg, "reached no refusal") {
		t.Fatalf("failed for the wrong reason: %s", f.msg)
	}
}

// TestTwoFixturesMayShareOneSiteWhenEverySiteIsStillCovered: A FIXTURE PER REFUSAL *SITE* IS NOT A FIXTURE PER REFUSAL *BOUNDARY*.
func TestTwoFixturesMayShareOneSiteWhenEverySiteIsStillCovered(t *testing.T) {
	src := []byte("sys.exit(3)\nsys.exit(4)\n")
	s := Spec{Tool: "s", Synthetic: true, RefusalSites: 2, Fixtures: []Fixture{
		{Name: "happy"},
		{Name: "absent", Refusal: "one", Break: func(string) error { return nil }},
		{Name: "unreadable", Refusal: "one", Break: func(string) error { return nil }},
		{Name: "other", Refusal: "two", Break: func(string) error { return nil }},
	}}
	f := &fake{}
	AssertRefusalSiteCount(f, s, src)
	if f.failed {
		t.Fatalf("rejected two boundaries of one site while both sites are covered: %s", f.msg)
	}
}

// TestAPassesFixtureThatStartsRefusingIsRejected: A FIXTURE THAT MUST *NOT* REFUSE IS THE OTHER HALF OF A BOUNDARY.
func TestAPassesFixtureThatStartsRefusingIsRejected(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	s.PythonArgs, s.GoArgs = []string{"bad"}, []string{"bad"}
	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{
		Name:   "at-the-boundary",
		Passes: true,
		Break:  func(string) error { return nil },
	}, mk)
	if !f.failed {
		t.Fatal("the harness accepted a must-not-refuse fixture that refused on both sides; " +
			"they agreed, so nothing but an explicit success assertion can see it")
	}
	if !strings.Contains(f.msg, "must not refuse") {
		t.Fatalf("failed for the wrong reason: %s", f.msg)
	}

	s2, mk2 := buildPair(t, pyRefusing, goFaithful)
	RunFixture(t, t.Context(), s2, Fixture{
		Name:   "at-the-boundary",
		Passes: true,
		Break:  func(string) error { return nil },
	}, mk2)
}

// TestAPassesFixtureIsNeitherHappyNorARefusalFixture: A Passes fixture is neither the happy fixture nor a refusal fixture.
func TestAPassesFixtureIsNeitherHappyNorARefusalFixture(t *testing.T) {
	src := []byte("sys.exit(3)\n")
	base := []Fixture{
		{Name: "happy"},
		{Name: "trips", Refusal: "one", Break: func(string) error { return nil }},
		{Name: "at-boundary", Passes: true, Break: func(string) error { return nil }},
	}
	f := &fake{}
	AssertRefusalSiteCount(f, Spec{Tool: "s", RefusalSites: 1, Fixtures: base}, src)
	if f.failed {
		t.Fatalf("rejected a valid set containing a Passes fixture: %s", f.msg)
	}

	bad := []Fixture{
		{Name: "happy"},
		{Name: "trips", Refusal: "one", Break: func(string) error { return nil }},
		{Name: "contradiction", Passes: true, Refusal: "one", Break: func(string) error { return nil }},
	}
	f2 := &fake{}
	AssertRefusalSiteCount(f2, Spec{Tool: "s", RefusalSites: 1, Fixtures: bad}, src)
	if !f2.failed {
		t.Fatal("accepted a fixture that both asserts success and names a refusal it must reach")
	}
}

// TestRefusalSitesCountsStringAndComputedExits: THE DEFINITION COUNTED ONLY NUMERIC EXITS, AND MOST OF THIS REPO'S REFUSALS ARE STRINGS.
func TestRefusalSitesCountsStringAndComputedExits(t *testing.T) {
	for name, tc := range map[string]struct {
		src  string
		want int
	}{
		"string argument, the dominant spelling here": {`sys.exit("REFUSING: empty")`, 1},
		"f-string argument":                           {"sys.exit(f\"REFUSING: only {n} keys\")", 1},
		"multi-line string argument":                  {"sys.exit(\"REFUSING: a very long \"\n    \"message\")", 1},
		"computed code":                               {"sys.exit(rc)", 1},
		"exit zero is not a refusal":                  {"sys.exit(0)", 0},
		"bare exit is not a refusal":                  {"sys.exit()", 0},
		"raise SystemExit with a string":              {`raise SystemExit("REFUSING: no")`, 1},
	} {
		if got := RefusalSites([]byte(tc.src)); got != tc.want {
			t.Errorf("%s: counted %d, want %d", name, got, tc.want)
		}
	}
}

// TestANoFixtureSiteCountsWithoutASandboxFixture: SOME REFUSALS GUARD THE IMPLEMENTATION, NOT THE INPUT, AND NO SANDBOX STATE REACHES THEM.
func TestANoFixtureSiteCountsWithoutASandboxFixture(t *testing.T) {
	src := []byte("sys.exit(3)\nsys.exit(4)\n")
	ok := Spec{Tool: "s", RefusalSites: 2, Fixtures: []Fixture{
		{Name: "happy"},
		{Name: "input", Refusal: "one", Break: func(string) error { return nil }},
		{Name: "self", Refusal: "two", NoFixture: true,
			Note: "guards the expander itself; covered by a mutation control"},
	}}
	f := &fake{}
	AssertRefusalSiteCount(f, ok, src)
	if f.failed {
		t.Fatalf("rejected a valid set containing a self-check site: %s", f.msg)
	}

	for name, fx := range map[string]Fixture{
		"no note":       {Name: "self", Refusal: "two", NoFixture: true},
		"no refusal":    {Name: "self", NoFixture: true, Note: "because"},
		"breaks anyway": {Name: "self", Refusal: "two", NoFixture: true, Note: "b", Break: func(string) error { return nil }},
	} {
		bad := Spec{Tool: "s", RefusalSites: 2, Fixtures: []Fixture{
			{Name: "happy"},
			{Name: "input", Refusal: "one", Break: func(string) error { return nil }},
			fx,
		}}
		f2 := &fake{}
		AssertRefusalSiteCount(f2, bad, src)
		if !f2.failed {
			t.Errorf("accepted a self-check fixture with %s", name)
		}
	}
}

// TestMoreFixturesThanSitesIsAllowedWhenOneSiteHasSeveralOutcomes: ONE LEXICAL EXIT CAN CARRY SEVERAL BEHAVIOURS, AND THE COUNT CHECK FORBADE COVERING THEM.
func TestMoreFixturesThanSitesIsAllowedWhenOneSiteHasSeveralOutcomes(t *testing.T) {
	src := []byte("sys.exit(main())\nsys.exit(3)\n")
	ok := Spec{Tool: "s", RefusalSites: 2, Fixtures: []Fixture{
		{Name: "happy"},
		{Name: "site-b", Refusal: "three", Break: func(string) error { return nil }},
		{Name: "funnel-no-file", Refusal: "set FILE=", Break: func(string) error { return nil }},
		{Name: "funnel-findings", Refusal: "flagged", Break: func(string) error { return nil }},
	}}
	f := &fake{}
	AssertRefusalSiteCount(f, ok, src)
	if f.failed {
		t.Fatalf("rejected a fixture set covering one site's several outcomes: %s", f.msg)
	}

	short := Spec{Tool: "s", RefusalSites: 3, Fixtures: []Fixture{
		{Name: "happy"},
		{Name: "only-one", Refusal: "three", Break: func(string) error { return nil }},
	}}
	f2 := &fake{}
	AssertRefusalSiteCount(f2, short, []byte("sys.exit(1)\nsys.exit(2)\nsys.exit(3)\n"))
	if !f2.failed {
		t.Fatal("accepted fewer distinct refusal messages than sites; a site has no fixture")
	}
}

const shRefusing = `#!/usr/bin/env bash
if [ "$1" = "bad" ]; then
  echo "REFUSING: bad" >&2
  exit 3
fi
echo "ok"
`

// TestAShellOriginalIsComparedTheSameWay: 19% OF THE LOC AND 73% OF THE REFUSALS ARE SHELL, AND THIS HARNESS COULD NOT RUN ONE.
func TestAShellOriginalIsComparedTheSameWay(t *testing.T) {
	s, mk := buildPair(t, shRefusing, goFaithful)
	s.Interpreter = "bash"
	s.PythonArgs, s.GoArgs = []string{"bad"}, []string{"bad"}
	RunFixture(t, t.Context(), s, Fixture{Name: "refusal", Refusal: "REFUSING: bad",
		Break: func(string) error { return nil }}, mk)

	s2, mk2 := buildPair(t, shRefusing, goWrongMessage)
	s2.Interpreter = "bash"
	s2.PythonArgs, s2.GoArgs = []string{"bad"}, []string{"bad"}
	f := &fake{}
	RunFixture(f, t.Context(), s2, Fixture{Name: "refusal"}, mk2)
	if !f.failed {
		t.Fatal("a shell original was compared without noticing a different refusal message")
	}
}

// TestSelfIdentitySweepsTheDeclaredVariable: PYTHONHASHSEED MEANS NOTHING TO bash, AND SWEEPING IT WOULD BE A GREEN TEST OF NOTHING.
func TestSelfIdentitySweepsTheDeclaredVariable(t *testing.T) {
	const localeSensitive = `#!/usr/bin/env bash
printf 'z\n_a\nab\nA\n' | sort | tr '\n' ' '
echo
`
	alt := ""
	for _, cand := range []string{"en_US.utf8", "en_GB.utf8", "lv_LV.utf8", "C.utf8"} {
		//nolint:gosec // `cand` is one of this test's own four literal locale names
		out, err := exec.CommandContext(t.Context(), "bash", "-c",
			"printf 'z\\n_a\\nab\\nA\\n' | LC_ALL="+cand+" sort | tr '\\n' ' '").Output()
		if err == nil && strings.TrimSpace(string(out)) != "A _a ab z" {
			alt = cand
			break
		}
	}
	if alt == "" {
		t.Skip("no installed locale collates differently from C; the sweep cannot discriminate here")
	}
	s, mk := buildPair(t, localeSensitive, goFaithful)
	s.Interpreter = "bash"
	s.SeedVar = "LC_ALL"
	f := &fake{}
	SelfIdentity(f, t.Context(), s, mk, []string{"C", alt, "C"})
	if !f.failed {
		t.Fatalf("LC_ALL=%s collates differently from C and the sweep did not notice", alt)
	}
	if !strings.Contains(f.msg, "LC_ALL") {
		t.Fatalf("failed without naming the swept variable: %s", f.msg)
	}
}

// TestRefusalSitesCountsShell: THE SITE COUNTER WAS PYTHON-ONLY, SO EVERY SHELL SPEC COUNTED ZERO.
func TestRefusalSitesCountsShell(t *testing.T) {
	for name, tc := range map[string]struct {
		src  string
		want int
	}{
		"exit nonzero":          {"exit 1\n", 1},
		"several":               {"exit 3\nexit 64\n", 2},
		"exit zero":             {"exit 0\n", 0},
		"bare exit":             {"exit\n", 0},
		"computed":              {"exit $rc\n", 1},
		"quoted computed":       {"exit \"$rc\"\n", 1},
		"return is not an exit": {"f() { return 1; }\n", 0},
	} {
		if got := ShellRefusalSites([]byte(tc.src)); got != tc.want {
			t.Errorf("%s: counted %d, want %d", name, got, tc.want)
		}
	}
	sh := Spec{Interpreter: "bash"}
	if got := SitesFor(sh, []byte("exit 3\n")); got != 1 {
		t.Errorf("a bash spec counted %d shell exits, want 1", got)
	}
	py := Spec{}
	if got := SitesFor(py, []byte("sys.exit(3)\n")); got != 1 {
		t.Errorf("a python spec counted %d, want 1", got)
	}
	if got := SitesFor(py, []byte("exit 3\n")); got != 0 {
		t.Errorf("a python spec counted a SHELL exit; the dispatch is not on the interpreter")
	}
}

// TestAnImpureSpecStillRunsItsPreEffectRefusals: AN IMPURE TOOL'S REFUSALS ARE PURE, AND REFUSING THE WHOLE SPEC THREW THEM AWAY.
func TestAnImpureSpecStillRunsItsPreEffectRefusals(t *testing.T) {
	s, mk := buildPair(t, pyRefusing, goFaithful)
	s.Impure = "creates a git worktree in the shared .git"
	s.PythonArgs, s.GoArgs = []string{"bad"}, []string{"bad"}

	RunFixture(t, t.Context(), s, Fixture{Name: "refusal", Refusal: "REFUSING: bad",
		BeforeEffects: true, Break: func(string) error { return nil }}, mk)

	f := &fake{}
	RunFixture(f, t.Context(), s, Fixture{Name: "happy"}, mk)
	if !f.failed {
		t.Fatal("an impure spec ran its happy fixture; the side effects were not prevented")
	}
	if !strings.Contains(f.msg, "impure") {
		t.Fatalf("failed for the wrong reason: %s", f.msg)
	}

	f2 := &fake{}
	RunFixture(f2, t.Context(), s, Fixture{Name: "undeclared", Refusal: "REFUSING: bad",
		Break: func(string) error { return nil }}, mk)
	if !f2.failed {
		t.Fatal("an impure spec ran a fixture that never declared it refuses before the effects")
	}
	if !strings.Contains(f2.msg, "BeforeEffects") {
		t.Fatalf("failed for the wrong reason: %s", f2.msg)
	}
}

// TestSandboxPathsAreNormalisedBeforeComparing: EACH IMPLEMENTATION GETS ITS OWN SANDBOX.
func TestSandboxPathsAreNormalisedBeforeComparing(t *testing.T) {
	const printsItsRoot = `import os, sys
print("ROOT " + os.getcwd())
print("ok")
`
	const goPrintsItsRoot = `package main

import (
	"fmt"
	"os"
)

func main() {
	wd, _ := os.Getwd()
	fmt.Println("ROOT " + wd)
	fmt.Println("ok")
}
`
	s, mk := buildPair(t, printsItsRoot, goPrintsItsRoot)
	RunFixture(t, t.Context(), s, Fixture{Name: "prints-its-own-root"}, mk)

	const goPrintsElsewhere = `package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	wd, _ := os.Getwd()
	fmt.Println("ROOT " + filepath.Join(wd, "somewhere-else"))
	fmt.Println("ok")
}
`
	s2, mk2 := buildPair(t, printsItsRoot, goPrintsElsewhere)
	f := &fake{}
	RunFixture(f, t.Context(), s2, Fixture{Name: "prints-a-different-path"}, mk2)
	if !f.failed {
		t.Fatal("normalisation hid a port that printed a different path")
	}
}

// TestGoldenKeyIgnoresTheVolatilePath: THE GOLDEN KEY MUST NOT MOVE BETWEEN RUNS, AND THE FIRST DRAFT'S DID.
func TestGoldenKeyIgnoresTheVolatilePath(t *testing.T) {
	base := Spec{
		Tool: "demo", Synthetic: true, PythonScript: "scripts/demo.py",
		PythonArgs: []string{"--check"}, GoArgs: []string{"demo", "--check"},
		Env: []string{"GOPOB_NICED=1", "PATH=/tmp/TestX111/bin:/usr/bin"},
	}
	other := base
	other.Env = []string{"GOPOB_NICED=1", "PATH=/tmp/TestX999/bin:/usr/bin"}
	f := Fixture{Name: "happy"}
	if a, b := goldenDir("/repo", base, f), goldenDir("/repo", other, f); a != b {
		t.Errorf("the golden key moved with PATH:\n  %s\n  %s", a, b)
	}
}

// TestGoldenKeySeparatesDifferentInvocations: BUT IT MUST STILL SEPARATE INVOCATIONS THAT EXPECT DIFFERENT BYTES.
func TestGoldenKeySeparatesDifferentInvocations(t *testing.T) {
	a := Spec{Tool: "demo", Synthetic: true, PythonScript: "s.py", PythonArgs: []string{"status"}}
	b := Spec{Tool: "demo", Synthetic: true, PythonScript: "s.py", PythonArgs: []string{"baseline"}}
	f := Fixture{Name: "happy"}
	if goldenDir("/repo", a, f) == goldenDir("/repo", b, f) {
		t.Error("two invocations with different arguments share one golden directory")
	}
	c := a
	c.Env = []string{"FORCE=1"}
	if goldenDir("/repo", a, f) == goldenDir("/repo", c, f) {
		t.Error("two invocations with different environments share one golden directory")
	}
}

// TestGoldenAbsenceIsOnlyToleratedBeforeAnyIsRecorded: "NOT RECORDED YET" AND "RECORDED, BUT THE KEY MOVED" LOOKED IDENTICAL.
func TestGoldenAbsenceIsOnlyToleratedBeforeAnyIsRecorded(t *testing.T) {
	repo := t.TempDir()
	s := Spec{Tool: "demo", Synthetic: true}
	f := Fixture{Name: "happy"}

	var probe fake
	assertGoldenPresent(&probe, repo, s, f, filepath.Join(repo, GoldenRoot, "demo", "happy-deadbeef"))
	if probe.failed {
		t.Error("a tool with no goldens at all should not fail; the rollout is incremental")
	}

	if err := os.MkdirAll(filepath.Join(repo, GoldenRoot, "demo", "happy-00000000"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, GoldenRoot, "demo", "happy-00000000", "exit"),
		[]byte("0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	probe = fake{}
	assertGoldenPresent(&probe, repo, s, f, filepath.Join(repo, GoldenRoot, "demo", "happy-deadbeef"))
	if !probe.failed {
		t.Error("a MISSED lookup against a tool that has goldens passed silently — which is the " +
			"exact failure the whole mechanism had before this guard existed")
	}
}

// TestGoldenRecordsAnUnwrittenFileAsAbsent: A DECLARED FILE THAT WAS NOT WRITTEN IS A RESULT.
func TestGoldenRecordsAnUnwrittenFileAsAbsent(t *testing.T) {
	sb := t.TempDir()
	if err := os.WriteFile(filepath.Join(sb, "present.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := collectGolden(sb, result{}, []string{"present.txt", "missing.txt"})
	if string(g.files["present.txt"]) != "x\n" {
		t.Errorf("present file recorded as %q", g.files["present.txt"])
	}
	if !g.absent["missing.txt"] {
		t.Error("an unwritten file must be recorded as ABSENT")
	}
	if _, ok := g.files["missing.txt"]; ok {
		t.Error("an absent file must not be recorded as CONTENT; a sentinel collides with real " +
			"bytes and, if it holds a NUL, makes the golden binary to git")
	}
}
