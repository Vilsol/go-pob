package difftest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const RecordEnv = "DIFFTEST_RECORD_GOLDEN"

const GoldenRoot = "scripts/gopob/testdata/golden"

func recording() bool { return os.Getenv(RecordEnv) != "" }

var (
	writtenMu sync.Mutex
	written   = map[string]string{}
)

func noteGoldenWrite(t Failer, dir, who string, g goldenSet) {
	t.Helper()
	h := sha256.New()
	fmt.Fprintf(h, "%d\x00", g.code)
	h.Write(g.stdout)
	h.Write([]byte{0})
	h.Write(g.stderr)
	for _, rel := range sortedKeys(g.files) {
		fmt.Fprintf(h, "\x00%s\x00", rel)
		h.Write(g.files[rel])
	}
	for _, rel := range sortedBoolKeys(g.absent) {
		fmt.Fprintf(h, "\x00absent:%s", rel)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	writtenMu.Lock()
	defer writtenMu.Unlock()
	if prev, ok := written[dir]; ok && prev != sum {
		t.Fatalf("two fixtures write DIFFERENT bytes to one golden directory:\n  %s\n  the second "+
			"is %s.\n  The key is (tool, fixture name, args, env, files) — two fixtures sharing all "+
			"of those must not expect different output, or whichever records last silently becomes "+
			"the expectation for both.", dir, who)
		return
	}
	written[dir] = sum
}

func sortedKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedBoolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func goldenDir(repo string, s Spec, f Fixture) string {
	h := sha256.New()
	for _, part := range [][]string{s.PythonArgs, s.GoArgs, stableEnv(s), s.EnvUnset, s.Files} {
		for _, v := range part {
			h.Write([]byte(v))
			h.Write([]byte{0})
		}
		h.Write([]byte{1})
	}
	h.Write([]byte(strconv.Itoa(int(s.Stderr))))
	h.Write([]byte(s.PythonScript))
	sum := hex.EncodeToString(h.Sum(nil))[:8]
	return filepath.Join(repo, GoldenRoot, s.Tool, f.Name+"-"+sum)
}

func goldenSpecNote(s Spec, f Fixture) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "tool     %s\n", s.Tool)
	fmt.Fprintf(&b, "fixture  %s\n", f.Name)
	fmt.Fprintf(&b, "original %s\n", s.PythonScript)
	fmt.Fprintf(&b, "args     %s\n", strings.Join(s.PythonArgs, " "))
	env := stableEnv(s)
	fmt.Fprintf(&b, "env      %s\n", strings.Join(env, " "))
	if len(s.EnvUnset) > 0 {
		fmt.Fprintf(&b, "unset    %s\n", strings.Join(s.EnvUnset, " "))
	}
	if len(s.Files) > 0 {
		fmt.Fprintf(&b, "files    %s\n", strings.Join(s.Files, " "))
	}
	return []byte(b.String())
}

func stableEnv(s Spec) []string {
	tmp := os.TempDir()
	var env []string
	for _, kv := range s.Env {
		if strings.HasPrefix(kv, "PATH=") {
			continue
		}
		if i := strings.IndexByte(kv, '='); i >= 0 && strings.Contains(kv[i+1:], tmp) {
			continue
		}
		env = append(env, kv)
	}
	sort.Strings(env)
	return env
}

func assertGoldenPresent(t Failer, repo string, s Spec, f Fixture, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, "exit")); err == nil {
		return
	}
	toolDir := filepath.Join(repo, GoldenRoot, s.Tool)
	entries, err := os.ReadDir(toolDir)
	if err != nil || len(entries) == 0 {
		return
	}
	t.Fatalf("%s/%s: %d golden(s) recorded for this tool, but NONE matches this fixture's key.\n"+
		"  looked for: %s\n"+
		"  Either the fixture is new — record with %s=1 — or the spec changed something the key is\n"+
		"  computed from and every golden for it has silently stopped being checked.",
		s.Tool, f.Name, len(entries), dir, RecordEnv)
}

const absentSuffix = ".difftest-absent"

type goldenSet struct {
	code   int
	stdout []byte
	stderr []byte
	files  map[string][]byte
	absent map[string]bool
}

func collectGolden(root string, r result, files []string) goldenSet {
	g := goldenSet{code: r.code, stdout: r.stdout, stderr: r.stderr,
		files: map[string][]byte{}, absent: map[string]bool{}}
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // the harness's own sandbox
		if err != nil {
			g.absent[rel] = true
			continue
		}
		g.files[rel] = bytes.ReplaceAll(b, []byte(root), []byte(rootToken))
	}
	return g
}

func writeGolden(dir string, g goldenSet, note []byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("golden: %w", err)
	}
	w := func(name string, b []byte) error {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o600); err != nil {
			return fmt.Errorf("golden %s: %w", name, err)
		}
		return nil
	}
	if err := w("spec.txt", note); err != nil {
		return err
	}
	if err := w("exit", []byte(strconv.Itoa(g.code)+"\n")); err != nil {
		return err
	}
	if err := w("stdout", g.stdout); err != nil {
		return err
	}
	if err := w("stderr", g.stderr); err != nil {
		return err
	}
	write := func(rel string, b []byte) error {
		dst := filepath.Join(dir, "files", rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return fmt.Errorf("golden: %w", err)
		}
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			return fmt.Errorf("golden %s: %w", rel, err)
		}
		return nil
	}
	for rel, b := range g.files {
		if err := write(rel, b); err != nil {
			return err
		}
	}
	for rel := range g.absent {
		if err := write(rel+absentSuffix, nil); err != nil {
			return err
		}
	}
	return nil
}

func checkGolden(t Failer, dir string, s Spec, f Fixture, g goldenSet) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(dir, "exit")) //nolint:gosec // the harness's own testdata
	if err != nil {
		return
	}
	wantCode, _ := strconv.Atoi(strings.TrimSpace(string(want)))
	if wantCode != g.code {
		t.Fatalf("%s/%s: exit code %d, but the ORIGINAL's recorded verdict is %d\n"+
			"  golden: %s\n  If this change is deliberate, re-record in a commit that does nothing "+
			"else and put the diff in the message.", s.Tool, f.Name, g.code, wantCode, dir)
		return
	}
	for _, c := range []struct {
		name string
		got  []byte
	}{{"stdout", g.stdout}, {"stderr", g.stderr}} {
		if c.name == "stderr" && s.Stderr == StderrFirstLine {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, c.name)) //nolint:gosec // the harness's own testdata
		if rerr != nil {
			continue
		}
		if c.name == "stderr" {
			b, c.got = s.normStderr(b), s.normStderr(c.got)
		}
		if !bytes.Equal(b, c.got) {
			t.Fatalf("%s/%s: %s differs from the ORIGINAL's recorded verdict\n"+
				"  golden: %s\n--- recorded ---\n%s\n--- now ---\n%s",
				s.Tool, f.Name, c.name, dir, b, c.got)
			return
		}
	}
	for rel, got := range g.files {
		if _, aerr := os.Stat(filepath.Join(dir, "files", rel+absentSuffix)); aerr == nil {
			t.Fatalf("%s/%s: %s WAS WRITTEN, but the ORIGINAL did not write it\n  golden: %s",
				s.Tool, f.Name, rel, dir)
			return
		}
		b, rerr := os.ReadFile(filepath.Join(dir, "files", rel)) //nolint:gosec // the harness's own testdata
		if rerr != nil {
			continue
		}
		if !bytes.Equal(b, got) {
			t.Fatalf("%s/%s: %s differs from the ORIGINAL's recorded verdict\n  golden: %s\n%s",
				s.Tool, f.Name, rel, dir, firstDifference(b, got))
			return
		}
	}
	for rel := range g.absent {
		if _, rerr := os.Stat(filepath.Join(dir, "files", rel)); rerr == nil {
			t.Fatalf("%s/%s: %s was NOT written, but the ORIGINAL wrote it\n  golden: %s",
				s.Tool, f.Name, rel, dir)
			return
		}
	}
}

func repoRootFor(s Spec) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("%s: locating the repo root: %w", s.Tool, err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%s: no go.mod above %s", s.Tool, dir)
		}
		dir = parent
	}
}

func firstDifference(want, got []byte) string {
	w, g := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return fmt.Sprintf("  first difference at line %d\n    recorded: %q\n    now:      %q\n"+
				"  (%d recorded lines, %d now)", i+1, a, b, len(w), len(g))
		}
	}
	return "  (line-identical; the difference is in trailing bytes)"
}
