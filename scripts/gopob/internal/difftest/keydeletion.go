package difftest

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func AssertKeyDeletionAgrees(t *testing.T, ctx context.Context, s Spec, jsonInputs []string,
	newSandbox func() (string, error),
) {
	if len(jsonInputs) == 0 {
		t.Fatalf("%s: AssertKeyDeletionAgrees was given no inputs; it would pass vacuously", s.Tool)
		return
	}
	total, diverged := 0, 0
	for _, rel := range jsonInputs {
		probe, err := newSandbox()
		if err != nil {
			t.Fatalf("%s: building the probe sandbox: %v", s.Tool, err)
			return
		}
		raw, err := os.ReadFile(filepath.Join(probe, rel)) //nolint:gosec // a sandbox path the caller named
		if err != nil {
			t.Fatalf("%s: %s is not readable in the sandbox: %v", s.Tool, rel, err)
			return
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("%s: %s is not valid JSON, so nothing can be deleted from it: %v",
				s.Tool, rel, err)
			return
		}
		paths := keyPaths(doc, nil)
		if len(paths) == 0 {
			t.Fatalf("%s: %s has no object keys; this sweep would pass vacuously", s.Tool, rel)
			return
		}
		for _, path := range paths {
			var cut any
			if err := json.Unmarshal(raw, &cut); err != nil {
				t.Fatalf("%s: re-decoding %s: %v", s.Tool, rel, err)
				return
			}
			if !deleteAt(cut, path) {
				continue
			}
			body, err := json.Marshal(cut)
			if err != nil {
				t.Fatalf("%s: re-encoding %s: %v", s.Tool, rel, err)
				return
			}
			total++
			if !keyDeletionAgrees(t, ctx, s, rel, body, strings.Join(path, "."), newSandbox) {
				diverged++
				break
			}
		}
	}
	t.Logf("%s: %d key deletions, %d divergent", s.Tool, total, diverged)
}

func keyDeletionAgrees(t *testing.T, ctx context.Context, s Spec, rel string, body []byte, label string,
	newSandbox func() (string, error),
) bool {
	py, err := newSandbox()
	if err != nil {
		t.Fatalf("%s: building the python sandbox: %v", s.Tool, err)
		return false
	}
	goSb, err := newSandbox()
	if err != nil {
		t.Fatalf("%s: building the go sandbox: %v", s.Tool, err)
		return false
	}
	for _, sb := range []string{py, goSb} {
		if err := os.WriteFile(filepath.Join(sb, rel), body, 0o600); err != nil {
			t.Fatalf("%s: writing the cut input: %v", s.Tool, err)
			return false
		}
	}
	rg := run(ctx, goSb, s.Env, s.EnvUnset, append([]string{s.GoBin}, s.GoArgs...), s.Stdin)

	if repo, rerr := repoRootFor(s); rerr == nil && originalsDeleted(repo) {
		for _, marker := range panicMarkers {
			if bytes.Contains(rg.stderr, marker) {
				t.Errorf("%s: with %s.%s DELETED the port PANICS — an unguarded read where the "+
					"original refused.\n  go exit %d: %s",
					s.Tool, rel, label, rg.code, lastLine(rg.stderr, rg.stdout))
				return false
			}
		}
		return true
	}

	rp := run(ctx, py, s.Env, s.EnvUnset,
		append([]string{s.interpreter(), filepath.Join(py, s.PythonScript)}, s.PythonArgs...), s.Stdin)
	if rp.code == rg.code && bytes.Equal(rp.stdout, rg.stdout) && bytes.Equal(rp.stderr, rg.stderr) {
		return true
	}
	t.Errorf("%s: with %s.%s DELETED the two implementations disagree — the original reads that "+
		"key without a guard, so it raises where the port takes a zero value."+
		"\n"+
		"  python exit %d: %s\n  go     exit %d: %s",
		s.Tool, rel, label, rp.code, lastLine(rp.stderr, rp.stdout), rg.code, lastLine(rg.stderr, rg.stdout))
	return false
}

func lastLine(streams ...[]byte) string {
	for _, b := range streams {
		lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
		if last := lines[len(lines)-1]; strings.TrimSpace(last) != "" {
			return last
		}
	}
	return "(no output)"
}

func keyPaths(node any, prefix []string) [][]string {
	var out [][]string
	switch v := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := append(append([]string{}, prefix...), k)
			out = append(out, p)
			out = append(out, keyPaths(v[k], p)...)
		}
	case []any:
		if len(v) > 0 {
			out = append(out, keyPaths(v[0], append(append([]string{}, prefix...), "[0]"))...)
		}
	}
	return out
}

func deleteAt(node any, path []string) bool {
	for _, step := range path[:len(path)-1] {
		switch v := node.(type) {
		case map[string]any:
			node = v[step]
		case []any:
			if step != "[0]" || len(v) == 0 {
				return false
			}
			node = v[0]
		default:
			return false
		}
	}
	m, ok := node.(map[string]any)
	if !ok {
		return false
	}
	last := path[len(path)-1]
	if _, ok := m[last]; !ok {
		return false
	}
	delete(m, last)
	return true
}
