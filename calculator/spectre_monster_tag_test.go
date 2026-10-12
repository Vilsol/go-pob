package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestHulkingMiscreationReachesConstructs covers the MonsterTag tag end to end.
func TestHulkingMiscreationReachesConstructs(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	for _, name := range []string{
		"0384-necromancer-animate-weapon-of-ranged-arms-7.xml",
		"0424-necromancer-animate-weapon-of-ranged-arms-8.xml",
	} {
		t.Run(name, func(t *testing.T) {
			rec, ok := oracle[name]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", name)
			}
			env := corpusEnvForLoadedData(t, name)
			if env.Minion == nil {
				t.Fatalf("REFUSING: %s builds no minion", name)
			}
			for _, key := range []string{"Speed", "CombinedDPS"} {
				want := rec.Minion[key]
				if want == 0 {
					t.Fatalf("REFUSING: PoB reports 0 for Minion.%s on %s", key, name)
				}
				if got := env.Minion.Output[key]; !oracleClose(got, want, oracleDumpTolerance) {
					t.Errorf("%s Minion.%s = %g, PoB %g (ratio %.4f)", name, key, got, want, got/want)
				}
			}
		})
	}
}

type monsterTagActor struct {
	tags []string
	ok   bool
}

func (a monsterTagActor) GetOutput(string) (float64, bool) { return 0, false }
func (a monsterTagActor) MonsterTags() ([]string, bool)    { return a.tags, a.ok }

// TestMonsterTagGate pins ModStore.lua:884-912.
func TestMonsterTagGate(t *testing.T) {
	for _, c := range []struct {
		name  string
		actor monsterTagActor
		neg   bool
		want  float64
	}{
		{"match, case-insensitive", monsterTagActor{[]string{"construct", "melee"}, true}, false, 10},
		{"no match", monsterTagActor{[]string{"undead"}, true}, false, 0},
		{"neg on no match", monsterTagActor{[]string{"undead"}, true}, true, 10},
		{"neg on match", monsterTagActor{[]string{"construct"}, true}, true, 0},
		{"no minionData", monsterTagActor{nil, false}, true, 0},
	} {
		db := moddb.NewModDB()
		db.Actor = c.actor
		db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 10).Tag(mod.MonsterTag("Construct").Neg(c.neg)))
		if got := db.Sum(mod.TypeIncrease, nil, "Damage"); got != c.want {
			t.Errorf("%s: Damage INC = %v, want %v", c.name, got, c.want)
		}
	}
}
