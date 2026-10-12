package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func cdrRealpathM(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	rest := []string{}
	cur := filepath.Clean(abs)
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			parts := append([]string{resolved}, rest...)
			return filepath.Join(parts...)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return filepath.Clean(abs)
		}
		rest = append([]string{filepath.Base(cur)}, rest...)
		cur = parent
	}
}

func corpusDumpRef(argv []string) int {
	ref, out := "", ""
	if len(argv) > 0 {
		ref = argv[0]
	}
	if len(argv) > 1 {
		out = argv[1]
	}
	if ref == "" || out == "" {
		fmt.Fprintln(os.Stderr, "usage: corpus_dump_ref.sh <ref> <out>")
		return 2
	}

	sha, ok, _ := awGit("", "rev-parse", "--verify", ref+"^{commit}")
	if !ok || sha == "" {
		fmt.Fprintf(os.Stderr, "not a commit: %s\n", ref)
		return 2
	}

	outAbs := cdrRealpathM(out)

	wt, err := os.MkdirTemp("", "go-pob-dumpref-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not check "+sha+" out into a temporary worktree")
		return 1
	}
	_ = os.Remove(wt)
	defer func() {
		_, _, _ = awGit("", "worktree", "remove", "--force", wt)
		_ = os.RemoveAll(wt)
	}()

	fmt.Printf("dumping %s (%s) -> %s\n", sha, ref, outAbs)
	if _, ok, errText := awGit("", "worktree", "add", "--detach", "--quiet", wt, sha); !ok {
		fmt.Fprint(os.Stderr, errText)
		fmt.Fprintf(os.Stderr, "could not check %s out into a temporary worktree\n", sha)
		return 1
	}

	cmd := exec.Command("go", "test", "-count=1", "-v", "./calculator/", "-run", "TestCorpusDump") //nolint:gosec,noctx // as in the original
	cmd.Dir = wt
	cmd.Env = append(os.Environ(), "DUMP="+outAbs)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	status := 0
	if err := cmd.Run(); err != nil {
		status = awExitStatus(err)
	}

	if status != 0 {
		fmt.Fprintf(os.Stderr, "dump of %s FAILED (go test exit %d) — %s may be absent or partial\n",
			sha, status, outAbs)
	}
	return status
}
