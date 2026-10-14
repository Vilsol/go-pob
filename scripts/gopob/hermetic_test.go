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
	os.Exit(m.Run())
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
