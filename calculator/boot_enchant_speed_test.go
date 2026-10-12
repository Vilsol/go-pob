package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
)

// TestKilledRecentlyBootEnchantIsFlagged pins ModParser.lua:3010-3013.
func TestKilledRecentlyBootEnchantIsFlagged(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS!=true")
	}
	if raw.LatestVersion != "3.20" {
		t.Skipf("SKIP: this test pins 3.20 behaviour but the process loaded %s game data", raw.LatestVersion)
	}

	const build = "0010-slayer-cyclone.xml"

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

	var flags []mod.MFlag
	for _, m := range env.ModDB.Mods["Speed"] {
		for _, tg := range m.Tags() {
			c, ok := tg.(*mod.ConditionTag)
			if ok && len(c.VarList) == 1 && c.VarList[0] == "KilledRecently" {
				flags = append(flags, m.Flags())
			}
		}
	}

	if len(flags) == 0 {
		t.Fatalf("REFUSING: %s produced no Condition:KilledRecently Speed mod at all — the "+
			"enchant did not parse, so this fixture cannot see the defect", build)
	}
	if len(flags) != 2 {
		t.Fatalf("%s: %d Condition:KilledRecently Speed mod(s) with flags %v, want 2 (one Attack, "+
			"one Cast). One unflagged mod matches every cfg, including passes with neither flag.",
			build, len(flags), flags)
	}
	var haveAttack, haveCast bool
	for _, f := range flags {
		haveAttack = haveAttack || f == mod.MFlagAttack
		haveCast = haveCast || f == mod.MFlagCast
	}
	if !haveAttack || !haveCast {
		t.Errorf("%s: flags %v, want exactly one Attack (%#x) and one Cast (%#x)",
			build, flags, mod.MFlagAttack, mod.MFlagCast)
	}
}
