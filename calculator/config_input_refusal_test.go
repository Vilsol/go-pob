package calculator

import (
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// TestConfigInstrumentsRefuseMalformedInput re-runs the config instruments in a child.
func TestConfigInstrumentsRefuseMalformedInput(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("cannot re-exec the test binary under wasm")
	}
	bin := os.Args[0]
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("cannot re-exec the test binary (%s): %s", bin, err)
	}
	inputs := []string{"CONFIG_FLIP_OUT", "CONFIG_OPTIONS", "POB_SRC", "LUAJIT", "FLIP_SCRIPT", "CONFIG_CENSUS_OUT", "CONFIG_PLACEHOLDERS"}
	base := slices.DeleteFunc(os.Environ(), func(kv string) bool {
		k, _, _ := strings.Cut(kv, "=")
		return slices.Contains(inputs, k)
	})
	base = append(base, "OPERATOR=true", "CORPUS=true")
	malformed := "testdata/config_input_malformed.json"
	scratch := t.TempDir()

	cases := []struct {
		test, task string
		bad        []string
	}{
		{"TestConfigFlip", "config-flip", []string{"CONFIG_FLIP_OUT=" + scratch, "CONFIG_OPTIONS=" + malformed,
			"POB_SRC=" + scratch, "LUAJIT=" + scratch, "FLIP_SCRIPT=" + scratch}},
		{"TestConfigVisibility", "config-visibility", []string{"CONFIG_FLIP_OUT=" + scratch, "CONFIG_OPTIONS=" + malformed}},
		{"TestConfigVisibility", "config-visibility", []string{"CONFIG_OPTIONS=" + malformed}},
		{"TestConfigOptionCensus", "config-census", []string{"CONFIG_OPTIONS=" + malformed, "CONFIG_CENSUS_OUT=" + scratch}},
		{"TestConfigPlaceholdersAgainstPoB", "config-placeholders", []string{"CONFIG_PLACEHOLDERS=" + malformed}},
	}
	run := func(test string, env []string) (string, error) {
		cmd := exec.CommandContext(t.Context(), bin, "-test.run=^"+test+"$", "-test.v", "-test.count=1") //nolint:gosec // re-runs this test binary
		cmd.Env = append(slices.Clone(base), env...)
		out, err := cmd.CombinedOutput()
		if !strings.Contains(string(out), "=== RUN   "+test) {
			t.Fatalf("%s did not run in the child:\n%s", test, out)
		}
		return string(out), err
	}
	for _, c := range cases {
		out, err := run(c.test, nil)
		if err != nil || !strings.Contains(out, "--- SKIP: "+c.test) || !strings.Contains(out, "`task "+c.task+"`") {
			t.Errorf("%s without input: want a SKIP naming `task %s`, got err=%v\n%s", c.test, c.task, err, out)
		}
		out, err = run(c.test, c.bad)
		if err == nil || !strings.Contains(out, "--- FAIL: "+c.test) {
			t.Errorf("%s with %v: want a FAIL, got err=%v\n%s", c.test, c.bad, err, out)
		}
	}
}
