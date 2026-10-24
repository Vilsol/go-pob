package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestMain: The developer's global git config leaks into every fixture repository and every tool under test.
func TestMain(m *testing.M) {
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_CONFIG_COUNT", "1")
	_ = os.Setenv("GIT_CONFIG_KEY_0", "maintenance.auto")
	_ = os.Setenv("GIT_CONFIG_VALUE_0", "false")
	code := m.Run()
	if portBinDir != "" {
		_ = os.RemoveAll(portBinDir)
	}
	os.Exit(code)
}

func TestFixtureCommitsStartNoBackgroundMaintenance(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	dir := t.TempDir()
	git := func(env []string, args ...string) string {
		t.Helper()
		//nolint:gosec // fixed git arguments this test wrote itself, against a temp repo
		c := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
		c.Env = append(append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t"), env...)
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	git(nil, "init", "-q", "-b", "main")
	trace := git([]string{"GIT_TRACE=1"}, "commit", "-q", "--allow-empty", "-m", "a")
	if !strings.Contains(trace, "trace: built-in: git commit") {
		t.Fatalf("GIT_TRACE printed no trace; this check cannot see a spawned command:\n%s", trace)
	}
	if strings.Contains(trace, "maintenance run") {
		t.Errorf("a fixture commit started background maintenance:\n%s", trace)
	}
}

const (
	probeSetarch    = `setarch "$(uname -m)" -R true`
	probeNamespaces = `setarch "$(uname -m)" -R unshare -rn -- true`
)

func requirePOSIX(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell: the fixtures run shell scripts and #! stubs, which Windows cannot execute")
	}
}

func requireWorks(t *testing.T, probe string) {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "sh", "-c", probe).CombinedOutput()
	if err != nil {
		t.Skipf("`%s` does not run on this machine (%v): %s", probe, err, strings.TrimSpace(string(out)))
	}
}
