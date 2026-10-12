package toolchain

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const AllowUnpinnedEnv = "GOPOB_ALLOW_UNPINNED"

var toolLine = regexp.MustCompile(`(?m)^\s*([a-z][\w-]*)\s*=\s*["']([^"']+)["']`)

func Pins(root string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(root, "mise.toml"))
	if err != nil {
		return nil, fmt.Errorf("REFUSING: mise.toml is not readable: %w", err)
	}
	out := map[string]string{}
	inTools := false
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			inTools = t == "[tools]"
			continue
		}
		if !inTools || strings.HasPrefix(t, "#") {
			continue
		}
		if m := toolLine.FindStringSubmatch(line); m != nil {
			out[m[1]] = m[2]
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("REFUSING: mise.toml declares no [tools], so nothing is pinned")
	}
	return out, nil
}

func HaveMise() bool { _, err := exec.LookPath("mise"); return err == nil }

func Command(ctx context.Context, root, tool string, args ...string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch {
	case HaveMise():
		//nolint:gosec // the tool name and args are the caller's own instrument invocation, not input
		cmd = exec.CommandContext(ctx, "mise", append([]string{"-C", root, "x", "--", tool}, args...)...)
	case os.Getenv(AllowUnpinnedEnv) != "":
		//nolint:gosec // same: this is a build tool spawning a build tool
		cmd = exec.CommandContext(ctx, tool, args...)
	default:
		return nil, fmt.Errorf(
			"REFUSING: mise is not on PATH, so %q would run UNPINNED and its answer would be "+
				"indistinguishable from a pinned one; set %s=1 to accept that", tool, AllowUnpinnedEnv)
	}
	cmd.Dir = root
	cmd.Env = ChildEnv(root, os.Environ())
	return cmd, nil
}

func ChildEnv(root string, base []string) []string {
	env := make([]string, 0, len(base)+2)
	for _, kv := range base {
		if strings.HasPrefix(kv, "GOPOB_NICED=") || strings.HasPrefix(kv, "GOTOOLCHAIN=") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "GOPOB_NICED=1")
	if pins, err := Pins(root); err == nil {
		if v, ok := pins["go"]; ok {
			env = append(env, "GOTOOLCHAIN=go"+v)
		}
	}
	return env
}

var goVersion = regexp.MustCompile(`go version go(\S+) `)

func VerifyGo(ctx context.Context, root string) error {
	pins, err := Pins(root)
	if err != nil {
		return err
	}
	want, ok := pins["go"]
	if !ok {
		return fmt.Errorf("REFUSING: mise.toml [tools] does not pin go")
	}
	cmd, err := Command(ctx, root, "go", "version")
	if err != nil {
		return err
	}
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("REFUSING: `go version` failed under the spawner: %w", err)
	}
	m := goVersion.FindStringSubmatch(string(out))
	if m == nil {
		return fmt.Errorf("REFUSING: could not parse %q as a go version", strings.TrimSpace(string(out)))
	}
	if m[1] != want {
		return fmt.Errorf("REFUSING: the spawner resolved go%s but mise.toml pins go%s; "+
			"go.mod declares a FLOOR and cannot see this", m[1], want)
	}
	return nil
}
