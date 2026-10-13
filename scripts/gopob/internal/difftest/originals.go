package difftest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	originalsOnce sync.Once
	originalsAway bool
)

func originalsDeleted(repo string) bool {
	originalsOnce.Do(func() {
		n := 0
		for _, pat := range []string{"scripts/*.py", "oracle/*.py"} {
			m, _ := filepath.Glob(filepath.Join(repo, pat))
			n += len(m)
		}
		originalsAway = n == 0
	})
	return originalsAway
}

func ReadOriginal(t Failer, repo, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repo, rel)) //nolint:gosec // a repo-relative path from a Spec
	if err == nil {
		return b
	}
	if originalsDeleted(repo) {
		return nil
	}
	t.Fatalf("%s is not readable, but other originals are still in the tree: %v\n"+
		"  A missing original is only expected once they are ALL gone. Until then this is a wrong "+
		"path, and returning nil would silently retire the differential for this tool.", rel, err)
	return nil
}

func OriginalsDeleted(repo string) bool { return originalsDeleted(repo) }

var panicMarkers = [][]byte{[]byte("panic: "), []byte("goroutine "), []byte("runtime error: ")}

func runFixtureGolden(t Failer, ctx context.Context, s Spec, f Fixture,
	newSandbox func() (string, error), repo string,
) {
	t.Helper()
	if s.Impure != "" && (f.Refusal == "" || !f.BeforeEffects) {
		return
	}
	sb, err := newSandbox()
	if err != nil {
		t.Fatalf("%s/%s: building the sandbox: %v", s.Tool, f.Name, err)
		return
	}
	if f.Break != nil {
		if err := f.Break(sb); err != nil {
			t.Fatalf("%s/%s: breaking the sandbox: %v", s.Tool, f.Name, err)
			return
		}
	}
	rg := run(ctx, sb, s.Env, s.EnvUnset, append([]string{s.GoBin}, s.GoArgs...), s.Stdin).normalise(sb)

	if f.Passes && rg.code != 0 {
		t.Fatalf("%s/%s must not refuse, and it exited %d\n--- stderr ---\n%s",
			s.Tool, f.Name, rg.code, rg.stderr)
		return
	}
	if f.Finding {
		if rg.code == 0 {
			t.Fatalf("%s/%s asserts a FINDING and exited 0 — the break no longer breaks anything",
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
				append([]string{s.GoBin}, s.GoArgs...), s.Stdin).normalise(clean)
			if rc.code == rg.code && bytes.Equal(rc.stdout, rg.stdout) && bytes.Equal(rc.stderr, rg.stderr) {
				t.Fatalf("%s/%s asserts a FINDING, but reports EXACTLY the same thing on an "+
					"unbroken sandbox — the Break has stopped breaking anything", s.Tool, f.Name)
				return
			}
		}
	}
	if f.Refusal != "" {
		stream, name := rg.stderr, "stderr"
		if s.RefusalOnStdout {
			stream, name = rg.stdout, "stdout"
		}
		if !bytes.Contains(stream, []byte(f.Refusal)) {
			t.Fatalf("%s/%s reached no refusal matching %q\n--- %s (exit %d) ---\n%s",
				s.Tool, f.Name, f.Refusal, name, rg.code, stream)
			return
		}
	}
	if f.Volatile != "" {
		return
	}
	dir := goldenDir(repo, s, f)
	if recording() {
		g := collectGolden(sb, rg, s.Files)
		noteGoldenWrite(t, dir, s.Tool+"/"+f.Name, g)
		if err := writeGolden(dir, g, goldenSpecNote(s, f)); err != nil {
			t.Fatalf("%s/%s: recording the golden: %v", s.Tool, f.Name, err)
		}
		return
	}
	assertGoldenPresent(t, repo, s, f, dir)
	checkGolden(t, dir, s, f, collectGolden(sb, rg, s.Files))
}

func selfIdentityGo(t Failer, ctx context.Context, s Spec, newSandbox func() (string, error), seeds []string, repo string) {
	t.Helper()
	filled, err := withGoInvocation(s, repo)
	if err != nil {
		t.Fatalf("%s: %v", s.Tool, err)
		return
	}
	s = filled
	var first result
	refusedEvery := true
	for i, seed := range seeds {
		sb, err := newSandbox()
		if err != nil {
			t.Fatalf("%s: building the sandbox: %v", s.Tool, err)
			return
		}
		r := run(ctx, sb, append(append([]string{}, s.Env...), s.seedVar()+"="+seed), s.EnvUnset,
			append([]string{s.GoBin}, s.GoArgs...), s.Stdin).normalise(sb)
		if r.code == 0 {
			refusedEvery = false
		}
		if i == 0 {
			first = r
			continue
		}
		if !bytes.Equal(first.stdout, r.stdout) || !bytes.Equal(first.stderr, r.stderr) || first.code != r.code {
			t.Fatalf("%s: %s=%s changes the output; the port is not stable against itself",
				s.Tool, s.seedVar(), seed)
			return
		}
	}
	if refusedEvery {
		t.Fatalf("%s: VACUOUS, not stable — every seeded run REFUSED (exit %d)", s.Tool, first.code)
	}
}

var (
	goBinOnce sync.Once
	goBinPath string
	errGoBin  error
)

func goBinary(repo string) (string, error) {
	goBinOnce.Do(func() {
		dir, err := os.MkdirTemp("", "difftest-port")
		if err != nil {
			errGoBin = fmt.Errorf("difftest: temp dir for the port: %w", err)
			return
		}
		bin := filepath.Join(dir, ExeName("gopob"))
		cmd := exec.Command("go", "build", "-o", bin, "./scripts/gopob") //nolint:gosec,noctx
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			errGoBin = fmt.Errorf("difftest: building the port: %w: %s", err, out)
			return
		}
		goBinPath = bin
	})
	return goBinPath, errGoBin
}

func withGoInvocation(s Spec, repo string) (Spec, error) {
	if s.GoBin == "" {
		bin, err := goBinary(repo)
		if err != nil {
			return s, err
		}
		s.GoBin = bin
	}
	if len(s.GoArgs) == 0 {
		s.GoArgs = append([]string{s.Tool}, s.PythonArgs...)
	}
	return s, nil
}
