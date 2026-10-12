package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestEnemyIsBossDefaultsToPinnacle pins ConfigOptions.lua:2060's `defaultIndex = 3`.
func TestEnemyIsBossDefaultsToPinnacle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	for _, k := range []string{"enemyArmour", "enemyEvasion", "enemyPhysicalDamage"} {
		if _, ok := env.ConfigPlaceholder[k]; !ok {
			t.Errorf("%s absent — an unset enemyIsBoss must take PoB's defaultIndex=3 "+
				"(Pinnacle), not fall through the switch writing nothing", k)
		}
	}

	const wantArmour = 61989
	if got := env.ConfigPlaceholder["enemyArmour"]; math.Abs(got-wantArmour) > 0.5 {
		t.Errorf("enemyArmour = %v, PoB dumps %d. A present-but-wrong value here means the "+
			"default resolved to the wrong branch, which no presence check would catch.",
			got, wantArmour)
	}
}

// TestUnsetEnemyIsBossAppliesThePinnacleMods covers the OTHER half of the same default.
func TestUnsetEnemyIsBossAppliesThePinnacleMods(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const build = "0001-necromancer-unknown.xml"
	env := corpusEnvForLoadedData(t, build)

	for _, in := range env.Build.Config.Inputs {
		if in.Name == "enemyIsBoss" {
			t.Fatalf("premise broken: %s now writes an enemyIsBoss <Input>, so it no longer "+
				"exercises the defaultIndex path", build)
		}
	}

	if got := env.Player.ModDB.Sum(mod.TypeBase, nil, "WarcryPower"); got != 20 {
		t.Errorf("WarcryPower BASE = %v, want 20 (ConfigOptions.lua:2132 on the Pinnacle branch); "+
			"0 means the handler never ran", got)
	}
	if !env.Enemy.ModDB.Flag(nil, "Condition:PinnacleBoss") {
		t.Error("Condition:PinnacleBoss is not set; the Pinnacle branch did not run")
	}
	if !env.Enemy.ModDB.Flag(nil, "Condition:RareOrUnique") {
		t.Error("Condition:RareOrUnique is not set; the Pinnacle branch did not run")
	}

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	rec, ok := oracle[build]
	if !ok {
		t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
	}
	want, ok := rec.Output["WarcryPower"]
	if !ok || want == 0 {
		t.Fatalf("REFUSING: PoB reports no nonzero WarcryPower for %s, so this cannot discriminate "+
			"against a handler that never ran", build)
	}
	if got := env.Player.Output["WarcryPower"]; math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
		t.Errorf("WarcryPower = %v, PoB reports %v", got, want)
	}
}
