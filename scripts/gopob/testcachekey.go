package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func tckFail(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "FAIL: "+format+"\n", args...)
	return 1
}

func tckProbe(gateTest string, extraEnv ...string) string {
	cmd := exec.Command("go", "test", "./calculator/", "-run", gateTest) //nolint:gosec,noctx // as in the original
	cmd.Env = append(append(os.Environ(), "CORPUS=true"), extraEnv...)
	out, _ := cmd.CombinedOutput()
	lines := pyUniversalLines(string(out))
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimRight(lines[len(lines)-1], "\n")
}

func testCacheKey(argv []string) int {
	_ = argv

	row, ok := tckAuthoritativeRow()
	if !ok {
		return 3
	}
	gateTest := tckField(row, 2)

	if err := exec.Command("go", "clean", "-testcache").Run(); err != nil { //nolint:gosec,noctx // as in the original
		return tckFail("could not clear the test cache")
	}

	cold := tckProbe(gateTest)
	switch {
	case strings.Contains(cold, "cached"):
		return tckFail("a freshly cleared cache still served a cached result: %s", cold)
	case strings.HasPrefix(cold, "ok"):
	default:
		fmt.Fprintf(os.Stderr, "REFUSING: the cold run neither passed nor cached: %s\n", cold)
		return 3
	}

	warm := tckProbe(gateTest)
	if !strings.Contains(warm, "cached") {
		return tckFail("an identical re-run was NOT cached, so this check cannot tell a correct " +
			"key from no\n       caching at all. Got: " + warm)
	}

	other := tckProbe(gateTest, "POE_VERSION=3.18") //go-only: retired-version-ok: a DIFFERENT version is the control
	if strings.Contains(other, "cached") {
		return tckFail("the 3.18 run was served from the 3.20 cache entry (%s).\n"+
			"       POE_VERSION is not in the cache key. See registerVersionInTestCacheKey — the "+
			"read must\n       happen while a test is RUNNING; a package init or a pre-m.Run "+
			"TestMain is too early.", other)
	}

	fmt.Println("test cache key OK: identical re-runs cache, POE_VERSION changes do not")
	return 0
}

func tckAuthoritativeRow() (string, bool) {
	cmd := exec.Command("bash", "-c", //nolint:gosec,noctx // sourcing the shim is what the original does
		`. scripts/gate_versions.sh && authoritative_row`)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimRight(string(out), "\n"), true
}

func tckField(row string, n int) string {
	if !strings.Contains(row, "|") {
		return row
	}
	fields := strings.Split(row, "|")
	if n < 1 || n > len(fields) {
		return ""
	}
	return fields[n-1]
}
