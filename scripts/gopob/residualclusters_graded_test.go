package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestResidualClustersGradedOverride pins the denominator every percentage in residual-clusters is computed against.
func TestResidualClustersGradedOverride(t *testing.T) {
	t.Parallel()
	repo := repoRoot(t)
	bin := buildPort(t, repo)
	sb := t.TempDir()

	var b strings.Builder
	b.WriteString("build\tstat\tclass\tpob\tours\n")
	for i := range 100 {
		b.WriteString("b" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".xml\tAlpha\twrong\t100\t130\n")
	}
	matrix := filepath.Join(sb, "matrix.tsv")
	if err := os.WriteFile(matrix, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	run := func(graded string) string {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), bin, "residual-clusters", "10") //nolint:gosec // a test-built binary
		cmd.Dir = sb
		cmd.Env = append(os.Environ(), "MATRIX="+matrix)
		if graded != "" {
			cmd.Env = append(cmd.Env, "GRADED="+graded)
		}
		out, err := cmd.CombinedOutput()
		if err != nil && graded != "bogus" {
			t.Fatalf("residual-clusters failed: %v\n%s", err, out)
		}
		return string(out)
	}

	if got := run("1000000"); !strings.Contains(got, "current gate ~ 99.9900%") {
		t.Fatalf("GRADED=1000000 with 100 wrong pairs must print 99.9900%%; got:\n%s", got)
	}
	if got := run(""); !strings.Contains(got, "current gate ~ 99.9969%") {
		t.Fatalf("with GRADED unset the fossil constant 3208353 must still be used; got:\n%s", got)
	}
	if got := run("bogus"); !strings.Contains(got, "REFUSING: GRADED must be a positive integer") {
		t.Fatalf("GRADED=bogus must refuse rather than silently using the constant; got:\n%s", got)
	}
}
