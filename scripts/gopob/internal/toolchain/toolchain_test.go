package toolchain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

func repo(t *testing.T) string {
	t.Helper()
	r, err := root.Resolve()
	if err != nil {
		t.Fatalf("cannot resolve the repo root: %v", err)
	}
	return r
}

func TestPinsReadsEveryToolFromMiseToml(t *testing.T) {
	pins, err := Pins(repo(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"go", "golangci-lint", "task", "luajit"} {
		if pins[want] == "" {
			t.Fatalf("mise.toml [tools] parsed without %q; got %v", want, pins)
		}
	}
	if strings.Contains(pins["go"], " ") || !strings.HasPrefix(pins["go"], "1.") {
		t.Fatalf("go pin looks wrong: %q", pins["go"])
	}
}

func TestPinsRefusesAnEmptyToolsTable(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "mise.toml"), []byte("[settings]\nexperimental = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Pins(d); err == nil {
		t.Fatal("an empty [tools] returned a map instead of refusing; every version assertion " +
			"built on it would then pass vacuously")
	}
}

// TestSpawnerResolvesThePinnedGoFromAnyWorkingDirectory: THE ONE THAT MATTERS.
func TestSpawnerResolvesThePinnedGoFromAnyWorkingDirectory(t *testing.T) {
	if !HaveMise() {
		t.Skip("mise is not installed; the pinning layer cannot be exercised")
	}
	r := repo(t)
	t.Chdir(t.TempDir())
	if err := VerifyGo(t.Context(), r); err != nil {
		t.Fatalf("the spawner did not resolve the pinned go: %v", err)
	}
}

// TestCommandRoutesThroughTheRepoRootNotTheCallersCwd: THE OUTCOME TEST ABOVE IS CARRIED BY TWO INDEPENDENT MECHANISMS AND CANNOT SEE EITHER ONE FAIL ALONE.
func TestCommandRoutesThroughTheRepoRootNotTheCallersCwd(t *testing.T) {
	if !HaveMise() {
		t.Skip("mise is not installed; there is no routing to assert")
	}
	r := repo(t)
	t.Chdir(t.TempDir())
	cmd, err := Command(t.Context(), r, "golangci-lint", "--version")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Dir != r {
		t.Fatalf("cmd.Dir is %q, want the repo root %q", cmd.Dir, r)
	}
	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "-C "+r) {
		t.Fatalf("argv does not pin mise to the repo root: %q", joined)
	}
}

func TestCommandRefusesRatherThanRunningUnpinned(t *testing.T) {
	if HaveMise() {
		t.Skip("mise is present, so the no-mise branch is not reachable here")
	}
	t.Setenv(AllowUnpinnedEnv, "")
	if _, err := Command(t.Context(), repo(t), "go", "version"); err == nil {
		t.Fatal("spawned an unpinned toolchain instead of refusing")
	}
}

// TestChildEnvExportsGopobNicedExactlyOnce: GOPOB_NICED MUST BE IN THE CHILD ENVIRONMENT.
func TestChildEnvExportsGopobNicedExactlyOnce(t *testing.T) {
	env := ChildEnv(repo(t), []string{"PATH=/usr/bin", "GOPOB_NICED=", "GOPOB_NICED=1"})
	n := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "GOPOB_NICED=") {
			n++
			if kv != "GOPOB_NICED=1" {
				t.Fatalf("GOPOB_NICED passed through as %q", kv)
			}
		}
	}
	if n != 1 {
		t.Fatalf("GOPOB_NICED appears %d times in the child environment, want exactly 1", n)
	}
}

// TestChildEnvPinsGotoolchainToTheMiseVersion: GOTOOLCHAIN covers the case mise.toml [env] cannot.
func TestChildEnvPinsGotoolchainToTheMiseVersion(t *testing.T) {
	r := repo(t)
	pins, err := Pins(r)
	if err != nil {
		t.Fatal(err)
	}
	want := "GOTOOLCHAIN=go" + pins["go"]
	for _, kv := range ChildEnv(r, []string{"GOTOOLCHAIN=go1.0.0"}) {
		if kv == want {
			return
		}
	}
	t.Fatalf("child environment does not carry %q", want)
}

func TestNiceIsIdempotentAndAbsolute(t *testing.T) {
	before, err := Niceness()
	if err != nil {
		t.Skipf("niceness unreadable on this platform: %v", err)
	}
	t.Setenv("GOPOB_NICED", "")
	t.Setenv("NICE", "7")
	got := Nice()
	after, _ := Niceness()
	if got != after {
		t.Fatalf("Nice() reported %d but the process is at %d", got, after)
	}
	if after != 7 && after != before {
		t.Fatalf("niceness is %d after Nice(); want 7 (or unchanged if the platform refuses)", after)
	}
	if got := Nice(); got != -1 {
		t.Fatalf("a second Nice() applied %d; it must stand down once GOPOB_NICED is set", got)
	}
	again, _ := Niceness()
	if again != after {
		t.Fatalf("niceness moved %d -> %d on a repeat call", after, again)
	}
}

// TestDeterministicLuaJITRefusesWhenItIsNotBuilt: A --version ASSERT CANNOT TELL THE TWO LUAJITS APART.
func TestDeterministicLuaJITRefusesWhenItIsNotBuilt(t *testing.T) {
	d := t.TempDir()
	_, err := LuaJITBinary(t.Context(), d, Deterministic)
	if err == nil {
		t.Fatal("returned a path for a deterministic luajit that does not exist")
	}
	if !strings.Contains(err.Error(), "task luajit-det") {
		t.Fatalf("the refusal does not name the remedy: %v", err)
	}
}

func TestLuaJITFingerprintsWhateverItHandsOver(t *testing.T) {
	r := repo(t)
	if _, err := os.Stat(filepath.Join(r, DeterministicPath)); err == nil {
		bin, err := LuaJITBinary(t.Context(), r, Deterministic)
		if err != nil {
			t.Fatal(err)
		}
		assertFingerprint(t, bin)
		t.Logf("deterministic luajit %s", bin)
		return
	}
	if !HaveMise() {
		t.Skip("neither a deterministic build nor mise is available")
	}
	bin, err := LuaJITBinary(t.Context(), r, Pinned)
	if err != nil {
		t.Skipf("mise could not resolve luajit here: %v", err)
	}
	assertFingerprint(t, bin)
	t.Logf("pinned luajit %s", bin)
}

func assertFingerprint(t *testing.T, bin Binary) {
	t.Helper()
	if len(bin.MD5) != 32 {
		t.Fatalf("md5 %q is not 32 hex chars", bin.MD5)
	}
	if bin.Size < 100_000 {
		t.Fatalf("%s is %d bytes — that is a stub or an empty file, not an interpreter",
			bin.Path, bin.Size)
	}
}

// TestNiceReportsTheNicenessItAchievedNotTheOneItRequested: Nice() MUST REPORT WHAT HAPPENED, NOT WHAT IT ASKED FOR.
func TestNiceReportsTheNicenessItAchievedNotTheOneItRequested(t *testing.T) {
	before, err := Niceness()
	if err != nil {
		t.Skipf("niceness unreadable on this platform: %v", err)
	}
	t.Setenv("GOPOB_NICED", "")
	t.Setenv("NICE", "7")
	got := Nice()
	actual, err := Niceness()
	if err != nil {
		t.Fatalf("cannot read back the niceness: %v", err)
	}
	if got != actual {
		t.Fatalf("Nice() reported %d, the process is actually at %d (started at %d) — the return "+
			"value is the REQUEST, not the result", got, actual, before)
	}
}
