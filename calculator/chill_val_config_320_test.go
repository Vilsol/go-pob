package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestChillValConfigIsBaseNotOverride pins ConfigOptions.lua:1342-1345.
func TestChillValConfigIsBaseNotOverride(t *testing.T) {
	enemyModList := moddb.NewModList()
	handler, ok := configurations["conditionEnemyChilledEffect"]
	if !ok {
		t.Fatal("REFUSING: no conditionEnemyChilledEffect handler — this test cannot answer")
	}
	handler(float64(15), moddb.NewModList(), enemyModList)

	var sawChillVal, sawDesired bool
	for _, m := range enemyModList.Mods() {
		switch m.Name() {
		case "ChillVal":
			sawChillVal = true
			if m.Type() == mod.TypeOverride {
				t.Errorf("ChillVal is OVERRIDE; PoB writes BASE (ConfigOptions.lua:1343). "+
					"An OVERRIDE is found by CalcPerform.lua:1056's self-chill lookup and "+
					"suppresses the default 10, which is what drove the enemy's ActionSpeed to "+
					"exactly -100 on 3143-occultist-reap. got type %v", m.Type())
			}
		case "DesiredChillVal":
			sawDesired = true
		}
	}
	if !sawChillVal {
		t.Error("no ChillVal mod emitted at all (ConfigOptions.lua:1343)")
	}
	if !sawDesired {
		t.Error("no DesiredChillVal mod emitted — ConfigOptions.lua:1344 writes it under the " +
			"NEGATED ChilledConfig condition, and it was missing entirely")
	}
}

// TestChillValSelfChillMatchesPoB is the corpus-anchored half: the enemy's summed INC ActionSpeed on the build that produced the +Inf must be PoB's -95, not -100.
func TestChillValSelfChillMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.20")
	env := corpusEnvForLoadedData(t, "3143-occultist-reap.xml")
	got := env.Enemy.ModDB.Sum(mod.TypeIncrease, nil, "ActionSpeed")
	const want = -95.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("enemy INC ActionSpeed = %v, want %v (PoB's dumped total: Chill -10, Chill -15, "+
			"Freeze -70). At exactly -100 CalcDefence.lua:2085's unguarded division yields +Inf.",
			got, want)
	}
}
