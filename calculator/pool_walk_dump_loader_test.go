package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const poolWalkDumpTemplate = `{"build":"%BUILD%","callCount":1,"truncated":false,` +
	`"inputs":{"EnergyShieldRecoveryCap":2069,"Life":2013,"LifeRecoverable":2013,` +
	`"ManaUnreserved":97,"TotalEHP":685646.96326674,"Ward":1147},` +
	`"calls":[{"n":1,"damageIn":%DAMAGE%,` +
	`"poolsIn":{"EnergyShield":2069,"Life":2013,"Mana":97,"Ward":1147},` +
	`"poolsOut":{"EnergyShield":2069,"Life":2013,"Mana":97,"Ward":0}}]}`

func writePoolWalkDump(t *testing.T, dir, name, damage string) string {
	t.Helper()
	body := strings.ReplaceAll(poolWalkDumpTemplate, "%BUILD%", name+".xml")
	body = strings.ReplaceAll(body, "%DAMAGE%", damage)
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("REFUSING: cannot build the fixture: %v", err)
	}
	return path
}

// TestPoolWalkDumpLoaderIsolatesOneBadFile covers the BLAST RADIUS half of the pool-walk instrument.
func TestPoolWalkDumpLoaderIsolatesOneBadFile(t *testing.T) {
	dir := t.TempDir()

	good := []string{
		writePoolWalkDump(t, dir, "0001-a", `{"Physical":82.07}`),
		writePoolWalkDump(t, dir, "0003-c", `{"Fire":93.89}`),
		writePoolWalkDump(t, dir, "0004-d", `{"Chaos":12}`),
	}
	bad := writePoolWalkDump(t, dir, "0002-b", `[]`)

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("REFUSING: fixture built %d files, want 4", len(files))
	}

	entries, failures := loadPoolWalkDumps(files)

	if len(entries) != len(good) {
		t.Errorf("loaded %d usable dumps, want %d -- one bad file must cost ONE build, not the run", len(entries), len(good))
	}
	seen := map[string]bool{}
	for _, e := range entries {
		seen[e.Build] = true
	}
	for _, g := range good {
		name := strings.TrimSuffix(filepath.Base(g), ".json") + ".xml"
		if !seen[name] {
			t.Errorf("%s was not loaded; a failure earlier in sort order swallowed it", name)
		}
	}

	if len(failures) != 1 {
		t.Fatalf("reported %d failures, want exactly 1 -- a dropped failure is a silent halving", len(failures))
	}
	if failures[0].File != bad {
		t.Errorf("failure names %s, want %s", failures[0].File, bad)
	}
	if failures[0].Reason == "" {
		t.Errorf("failure carries no reason; %q must say WHY it could not be used", bad)
	}
}

// TestPoolWalkEmptyDamageTableIsDataNotCorruption pins the distinction the whole fix rests on.
func TestPoolWalkEmptyDamageTableIsDataNotCorruption(t *testing.T) {
	dir := t.TempDir()

	object := writePoolWalkDump(t, dir, "0001-object", `{}`)
	entries, failures := loadPoolWalkDumps([]string{object})
	if len(failures) != 0 {
		t.Fatalf("an empty damageIn OBJECT was rejected (%v); it is legitimate data", failures)
	}
	if len(entries) != 1 || len(entries[0].Calls) != 1 {
		t.Fatalf("REFUSING: fixture did not produce one call")
	}
	if entries[0].Calls[0].DamageIn == nil {
		t.Errorf("damageIn came back nil; the zero-damage probe must be COMPARED, and a nil map " +
			"is indistinguishable from a call the dumper failed to record")
	}
	if got := len(entries[0].Calls[0].DamageIn); got != 0 {
		t.Errorf("damageIn holds %d entries, want 0", got)
	}

	array := writePoolWalkDump(t, dir, "0002-array", `[]`)
	_, failures = loadPoolWalkDumps([]string{array})
	if len(failures) != 1 {
		t.Errorf("an empty damageIn ARRAY was accepted; it must stay a loud failure so the "+
			"emitter is what gets fixed (got %d failures)", len(failures))
	}
}
