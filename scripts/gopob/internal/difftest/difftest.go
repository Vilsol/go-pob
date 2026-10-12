package difftest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Failer interface {
	Helper()
	Fatalf(format string, args ...any)
}

type StderrContract int

const (
	StderrExact StderrContract = iota
	StderrFirstLine
)

type Spec struct {
	Tool            string
	PythonScript    string
	Interpreter     string
	SeedVar         string
	PythonArgs      []string
	GoBin           string
	GoArgs          []string
	Env             []string
	EnvUnset        []string
	Stdin           []byte
	Stderr          StderrContract
	StderrNormalise func([]byte) []byte
	Files           []string
	Impure          string
	RefusalSites    int
	RefusalOnStdout bool
	Library         []string
	Synthetic       bool
	Fixtures        []Fixture
}

func (s Spec) normStderr(b []byte) []byte {
	if s.StderrNormalise == nil {
		return b
	}
	return s.StderrNormalise(b)
}

type Fixture struct {
	Name            string
	Break           func(sandbox string) error
	BeforeEffects   bool
	NoFixture       bool
	OriginalRetired bool
	Note            string
	Passes          bool
	Finding         bool
	Volatile        string
	Refusal         string
}

func (s Spec) interpreter() string {
	if s.Interpreter == "" {
		return "python3"
	}
	return s.Interpreter
}

func (s Spec) seedVar() string {
	if s.SeedVar == "" {
		return "PYTHONHASHSEED"
	}
	return s.SeedVar
}

type result struct {
	stdout, stderr []byte
	code           int
}

func (r result) lines(b []byte) int { return bytes.Count(b, []byte("\n")) }

const rootToken = "<sandbox>"

func (r result) normalise(dir string) result {
	tok := []byte(rootToken)
	sub := func(b []byte) []byte {
		b = bytes.ReplaceAll(b, []byte(dir), tok)
		if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved != dir {
			b = bytes.ReplaceAll(b, []byte(resolved), tok)
		}
		return b
	}
	r.stdout, r.stderr = sub(r.stdout), sub(r.stderr)
	return r
}

var (
	pyExitCall = regexp.MustCompile(`(?:sys\.exit|raise SystemExit)\s*\(`)
	pyExitZero = regexp.MustCompile(`(?:sys\.exit|raise SystemExit)\s*\(\s*0?\s*\)`)
)

var shExit = regexp.MustCompile(`\bexit\s+(?:[1-9][0-9]*\b|"?\$)`)

func RefusalSites(src []byte) int {
	return len(pyExitCall.FindAllIndex(src, -1)) - len(pyExitZero.FindAllIndex(src, -1))
}

func ShellRefusalSites(src []byte) int { return len(shExit.FindAllIndex(src, -1)) }

func SitesFor(s Spec, src []byte) int {
	if strings.HasPrefix(s.interpreter(), "bash") || strings.HasPrefix(s.interpreter(), "sh") {
		return ShellRefusalSites(src)
	}
	return RefusalSites(src)
}

func AssertRefusalSiteCount(t Failer, s Spec, pythonSrc []byte) {
	t.Helper()
	found := SitesFor(s, pythonSrc)
	if pythonSrc == nil {
		found = s.RefusalSites
	}
	if found != s.RefusalSites {
		t.Fatalf("%s: source has %d refusal site(s), the spec declares %d — a site was added or "+
			"removed and its fixture was not", s.Tool, found, s.RefusalSites)
	}
	named := map[string]bool{}
	happy := 0
	for _, f := range s.Fixtures {
		if f.NoFixture {
			switch {
			case f.Refusal == "":
				t.Fatalf("%s: self-check fixture %q names no refusal site", s.Tool, f.Name)
			case f.Note == "":
				t.Fatalf("%s: self-check fixture %q gives no reason; a self-check without a "+
					"stated reason is a missing fixture with better manners", s.Tool, f.Name)
			case f.Break != nil:
				t.Fatalf("%s: self-check fixture %q has a Break; if a sandbox state reaches the "+
					"site it is not a self-check", s.Tool, f.Name)
			}
			named[f.Refusal] = true
			continue
		}
		if f.Break == nil {
			happy++
			continue
		}
		if f.Passes || f.Finding {
			if f.Refusal != "" {
				t.Fatalf("%s: fixture %q asserts success and also names refusal %q; it cannot "+
					"assert both outcomes", s.Tool, f.Name, f.Refusal)
			}
			if f.Passes && f.Finding {
				t.Fatalf("%s: fixture %q asserts both a clean run and a finding", s.Tool, f.Name)
			}
			continue
		}
		if f.Refusal == "" {
			t.Fatalf("%s: fixture %q breaks the sandbox but names no refusal site", s.Tool, f.Name)
		}
		named[f.Refusal] = true
	}
	if happy != 1 {
		t.Fatalf("%s: %d happy fixtures, want exactly 1", s.Tool, happy)
	}
	if len(named) < s.RefusalSites {
		t.Fatalf("%s: %d distinct refusal message(s) across the fixtures for %d refusal site(s); "+
			"at least one negative fixture per site is the contract, and fixtures sharing a "+
			"message count once", s.Tool, len(named), s.RefusalSites)
	}
}

func dropEnv(env []string, names []string) []string {
	if len(names) == 0 {
		return env
	}
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[n] = true
	}
	out := env[:0:0]
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i < 0 || !drop[kv[:i]] {
			out = append(out, kv)
		}
	}
	return out
}

func run(ctx context.Context, dir string, env, unset []string, argv []string, stdin []byte) result {
	//nolint:gosec // the argv is the harness's own fixture invocation
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = dropEnv(append(os.Environ(), env...), unset)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	_ = cmd.Run()
	return result{stdout: out.Bytes(), stderr: errb.Bytes(), code: cmd.ProcessState.ExitCode()}
}

func firstLine(b []byte) []byte {
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		return b[:i]
	}
	return b
}

func RunFixture(t Failer, ctx context.Context, s Spec, f Fixture, newSandbox func() (string, error)) {
	t.Helper()
	if f.OriginalRetired {
		if f.Note == "" {
			t.Fatalf("%s: fixture %q declares OriginalRetired and gives no reason; a comparison "+
				"declared void without a stated reason is an untested path with better manners",
				s.Tool, f.Name)
			return
		}
		fmt.Fprintf(os.Stderr,
			"%s/%s: COMPARISON VOID — the original's dependency was deleted by the fold, so the "+
				"two sides can only agree if the port keeps calling a file that does not exist.\n"+
				"    %s\n", s.Tool, f.Name, f.Note)
		return
	}
	if s.Impure != "" {
		switch {
		case f.Refusal == "":
			t.Fatalf("%s declares itself impure (%s); only a REFUSAL fixture may run against it, "+
				"and %q does not refuse", s.Tool, s.Impure, f.Name)
			return
		case !f.BeforeEffects:
			t.Fatalf("%s is impure (%s) and fixture %q does not declare BeforeEffects; the harness "+
				"cannot tell whether this refusal fires before the side effects, so the author "+
				"must", s.Tool, s.Impure, f.Name)
			return
		}
	}
	py, err := newSandbox()
	if err != nil {
		t.Fatalf("%s/%s: building the python sandbox: %v", s.Tool, f.Name, err)
		return
	}
	if !s.Synthetic {
		if repo, rerr := repoRootFor(s); rerr == nil && originalsDeleted(repo) {
			runFixtureGolden(t, ctx, s, f, newSandbox, repo)
			return
		}
	}
	goSb, err := newSandbox()
	if err != nil {
		t.Fatalf("%s/%s: building the go sandbox: %v", s.Tool, f.Name, err)
		return
	}
	if f.Break != nil {
		if err := f.Break(py); err != nil {
			t.Fatalf("%s/%s: breaking the python sandbox: %v", s.Tool, f.Name, err)
			return
		}
		if err := f.Break(goSb); err != nil {
			t.Fatalf("%s/%s: breaking the go sandbox: %v", s.Tool, f.Name, err)
			return
		}
	}
	rp := run(ctx, py, s.Env, s.EnvUnset, append([]string{s.interpreter(), filepath.Join(py, s.PythonScript)}, s.PythonArgs...), s.Stdin)
	rg := run(ctx, goSb, s.Env, s.EnvUnset, append([]string{s.GoBin}, s.GoArgs...), s.Stdin)

	rp = rp.normalise(py)
	rg = rg.normalise(goSb)

	if rp.code != rg.code {
		t.Fatalf("%s/%s exit code: python=%d go=%d\npython stderr:\n%s\ngo stderr:\n%s",
			s.Tool, f.Name, rp.code, rg.code, rp.stderr, rg.stderr)
		return
	}
	if !bytes.Equal(rp.stdout, rg.stdout) {
		t.Fatalf("%s/%s stdout differs\n--- python ---\n%s\n--- go ---\n%s",
			s.Tool, f.Name, rp.stdout, rg.stdout)
		return
	}
	switch s.Stderr {
	case StderrExact:
		if !bytes.Equal(s.normStderr(rp.stderr), s.normStderr(rg.stderr)) {
			t.Fatalf("%s/%s stderr differs\n--- python ---\n%s\n--- go ---\n%s",
				s.Tool, f.Name, rp.stderr, rg.stderr)
			return
		}
	case StderrFirstLine:
		if !bytes.Equal(firstLine(rp.stderr), firstLine(rg.stderr)) {
			t.Fatalf("%s/%s stderr first line differs\npython: %s\ngo:     %s",
				s.Tool, f.Name, firstLine(rp.stderr), firstLine(rg.stderr))
			return
		}
	}
	if f.Passes && rp.code != 0 {
		t.Fatalf("%s/%s must not refuse, and both implementations exited %d — a boundary fixture "+
			"on the accepting side of a guard now trips it\n--- stderr ---\n%s",
			s.Tool, f.Name, rp.code, rp.stderr)
		return
	}
	if f.Finding {
		if rp.code == 0 {
			t.Fatalf("%s/%s asserts a FINDING and both implementations exited 0 — the break no "+
				"longer breaks anything, so this fixture covers the happy path under another name",
				s.Tool, f.Name)
			return
		}
		if f.Break != nil {
			clean, cerr := newSandbox()
			if cerr != nil {
				t.Fatalf("%s/%s: building the control sandbox: %v", s.Tool, f.Name, cerr)
				return
			}
			rc := run(ctx, clean, s.Env, s.EnvUnset,
				append([]string{s.interpreter(), filepath.Join(clean, s.PythonScript)}, s.PythonArgs...),
				s.Stdin).normalise(clean)
			if rc.code == rp.code && bytes.Equal(rc.stdout, rp.stdout) && bytes.Equal(rc.stderr, rp.stderr) {
				t.Fatalf("%s/%s asserts a FINDING, but the tool reports EXACTLY the same thing on "+
					"an unbroken sandbox — the Break has stopped breaking anything and this "+
					"fixture covers nothing", s.Tool, f.Name)
				return
			}
		}
	}

	if f.Refusal != "" {
		stream, name := rp.stderr, "stderr"
		if s.RefusalOnStdout {
			stream, name = rp.stdout, "stdout"
		}
		if !bytes.Contains(stream, []byte(f.Refusal)) {
			t.Fatalf("%s/%s reached no refusal matching %q — the two implementations agree, but "+
				"this fixture no longer exercises the site it claims, so it covers nothing"+
				"\n--- %s (both sides, exit %d) ---\n%s",
				s.Tool, f.Name, f.Refusal, name, rp.code, stream)
			return
		}
	}

	if rp.lines(rp.stdout) != rg.lines(rg.stdout) {
		t.Fatalf("%s/%s stdout line counts differ: python=%d go=%d", s.Tool, f.Name,
			rp.lines(rp.stdout), rg.lines(rg.stdout))
		return
	}
	if s.Stderr == StderrExact && rp.lines(s.normStderr(rp.stderr)) != rg.lines(s.normStderr(rg.stderr)) {
		t.Fatalf("%s/%s stderr line counts differ: python=%d go=%d", s.Tool, f.Name,
			rp.lines(rp.stderr), rg.lines(rg.stderr))
		return
	}
	for _, rel := range s.Files {
		a, errA := os.ReadFile(filepath.Join(py, rel))
		b, errB := os.ReadFile(filepath.Join(goSb, rel))
		if (errA == nil) != (errB == nil) {
			t.Fatalf("%s/%s %s: python err=%v, go err=%v", s.Tool, f.Name, rel, errA, errB)
			return
		}
		if errA == nil && !bytes.Equal(a, b) {
			t.Fatalf("%s/%s %s: the two implementations wrote different bytes (%d vs %d)\n%s",
				s.Tool, f.Name, rel, len(a), len(b), firstDifference(a, b))
			return
		}
	}

	if f.Volatile != "" {
		return
	}
	repo, rerr := repoRootFor(s)
	if rerr != nil {
		return
	}
	dir := goldenDir(repo, s, f)
	if recording() {
		g := collectGolden(py, rp, s.Files)
		noteGoldenWrite(t, dir, s.Tool+"/"+f.Name, g)
		if err := writeGolden(dir, g, goldenSpecNote(s, f)); err != nil {
			t.Fatalf("%s/%s: recording the golden: %v", s.Tool, f.Name, err)
		}
		return
	}
	assertGoldenPresent(t, repo, s, f, dir)
	checkGolden(t, dir, s, f, collectGolden(goSb, rg, s.Files))
}

func SelfIdentity(t Failer, ctx context.Context, s Spec, newSandbox func() (string, error), seeds []string) {
	t.Helper()
	if repo, rerr := repoRootFor(s); rerr == nil && originalsDeleted(repo) && !s.Synthetic {
		selfIdentityGo(t, ctx, s, newSandbox, seeds, repo)
		return
	}
	if len(seeds) < 3 {
		t.Fatalf("%s: %d seeds is not a sweep; use at least 3", s.Tool, len(seeds))
		return
	}
	var first result
	refusedEvery := true
	for i, seed := range seeds {
		sb, err := newSandbox()
		if err != nil {
			t.Fatalf("%s: building a sandbox: %v", s.Tool, err)
			return
		}
		r := run(ctx, sb, append(append([]string{}, s.Env...), s.seedVar()+"="+seed), s.EnvUnset,
			append([]string{s.interpreter(), filepath.Join(sb, s.PythonScript)}, s.PythonArgs...),
			s.Stdin).normalise(sb)
		if r.code == 0 {
			refusedEvery = false
		}
		if i == 0 {
			first = r
			continue
		}
		if !bytes.Equal(first.stdout, r.stdout) || !bytes.Equal(first.stderr, r.stderr) || first.code != r.code {
			t.Fatalf("%s: %s=%s changes the output; it is not stable against itself, "+
				"so a byte-compare against Go would be measuring the seed", s.Tool, s.seedVar(), seed)
			return
		}
	}
	if refusedEvery {
		t.Fatalf("%s: VACUOUS, not stable — every seeded run REFUSED (exit %d), so the sweep "+
			"compared refusals rather than output", s.Tool, first.code)
	}
}

func CopyTree(dst, src string) error {
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return fmt.Errorf("sandbox copy: %w", err)
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			//nolint:gosec // both paths are harness-owned temp dirs, not attacker input
			return os.MkdirAll(target, 0o750)
		}
		b, err := os.ReadFile(p) //nolint:gosec // ditto
		if err != nil {
			return fmt.Errorf("sandbox copy: %w", err)
		}
		//nolint:gosec // ditto
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fmt.Errorf("sandbox copy: %w", err)
		}
		//nolint:gosec // ditto
		return os.WriteFile(target, b, info.Mode().Perm())
	})
	if err != nil {
		return fmt.Errorf("sandbox copy from %s: %w", src, err)
	}
	return nil
}

func Describe(s Spec) string {
	return fmt.Sprintf("%s: %d refusal site(s), %d fixture(s), stderr=%s", s.Tool, s.RefusalSites,
		len(s.Fixtures), map[StderrContract]string{StderrExact: "exact", StderrFirstLine: "first-line"}[s.Stderr])
}
