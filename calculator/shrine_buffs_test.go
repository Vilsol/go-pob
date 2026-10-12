package calculator

import (
	"math"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func shrineActor(t *testing.T) *Actor {
	t.Helper()
	db := moddb.NewModDB()
	initModDB(&Environment{}, db)
	if !db.HasMod(mod.TypeFlag, nil, "AccelerationShrine") {
		t.Fatal("REFUSING: initModDB no longer seeds the shrine flags — this fixture would test nothing")
	}
	return &Actor{ModDB: db, Output: map[string]float64{}}
}

// TestShrineBuffsApplyTheirMods pins CalcPerform.lua:290-374.
func TestShrineBuffsApplyTheirMods(t *testing.T) {
	quiet := shrineActor(t)
	applyShrineBuffs(quiet)
	if got := quiet.ModDB.Sum(mod.TypeIncrease, nil, "ProjectileSpeed"); got != 0 {
		t.Fatalf("REFUSING: a build with no shrine condition already has ProjectileSpeed %v — "+
			"the flags are not gated and every assertion below is meaningless", got)
	}

	a := shrineActor(t)
	a.ModDB.Conditions["AccelerationShrine"] = true
	applyShrineBuffs(a)
	if got := a.ModDB.Sum(mod.TypeIncrease, nil, "ActionSpeed"); got != 50 {
		t.Errorf("Acceleration Shrine ActionSpeed = %v, want 50", got)
	}
	if got := a.ModDB.Sum(mod.TypeIncrease, nil, "ProjectileSpeed"); got != 80 {
		t.Errorf("Acceleration Shrine ProjectileSpeed = %v, want 80", got)
	}

	scaled := shrineActor(t)
	scaled.ModDB.Conditions["ResistanceShrine"] = true
	scaled.ModDB.AddMod(mod.NewFloat("ShrineBuffEffect", mod.TypeIncrease, 25).Source("Test"))
	applyShrineBuffs(scaled)
	if got := scaled.ModDB.Sum(mod.TypeBase, nil, "ElementalResistMax"); got != 12 {
		t.Errorf("Resistance Shrine ElementalResistMax with +25%% shrine effect = %v, want 12 "+
			"(m_floor(10 * 1.25)) — the effect mod or its floor is missing", got)
	}

	repl := shrineActor(t)
	repl.ModDB.Conditions["ReplenishingShrine"] = true
	repl.ModDB.AddMod(mod.NewFloat("ShrineBuffEffect", mod.TypeIncrease, 10).Source("Test"))
	applyShrineBuffs(repl)
	if got := repl.ModDB.Sum(mod.TypeBase, nil, "LifeRegenPercent"); math.Abs(got-6.7*1.1) > 1e-9 {
		t.Errorf("Replenishing Shrine LifeRegenPercent = %v, want %v — :335 is a bare product, not a floor", got, 6.7*1.1)
	}
}

// TestLesserShrinesAreSuppressedByTheirGreaterTwin pins where the suppression actually lives, and it is NOT where CalcPerform suggests.
func TestLesserShrinesAreSuppressedByTheirGreaterTwin(t *testing.T) {
	lesser := shrineActor(t)
	lesser.ModDB.Conditions["LesserAccelerationShrine"] = true
	applyShrineBuffs(lesser)
	if got := lesser.ModDB.Sum(mod.TypeIncrease, nil, "ActionSpeed"); got != 10 {
		t.Fatalf("REFUSING: Lesser Acceleration Shrine alone gives ActionSpeed %v, want 10 — the "+
			"control is not reaching the block", got)
	}

	for _, tc := range []struct {
		lesser, greater, stat string
		modType               mod.Type
		want                  float64
	}{
		{"LesserAccelerationShrine", "AccelerationShrine", "ActionSpeed", mod.TypeIncrease, 50},
		{"LesserBrutalShrine", "BrutalShrine", "Damage", mod.TypeIncrease, 50},
		{"LesserMassiveShrine", "MassiveShrine", "AreaOfEffect", mod.TypeIncrease, 40},
		{"LesserImpenetrableShrine", "ImpenetrableShrine", "Armour", mod.TypeIncrease, 100},
		{"LesserResistanceShrine", "ResistanceShrine", "ElementalResist", mod.TypeBase, 50},
	} {
		t.Run(tc.lesser, func(t *testing.T) {
			a := shrineActor(t)
			a.ModDB.Conditions[tc.lesser] = true
			a.ModDB.Conditions[tc.greater] = true
			applyShrineBuffs(a)
			if got := a.ModDB.Sum(tc.modType, nil, tc.stat); got != tc.want {
				t.Errorf("%s with both shrines = %v, want %v (the greater alone) — the two stacked",
					tc.stat, got, tc.want)
			}
		})
	}
}
