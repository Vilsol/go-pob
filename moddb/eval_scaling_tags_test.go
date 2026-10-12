package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

type stubActor struct {
	outputs map[string]float64
}

func (a stubActor) GetOutput(name string) (float64, bool) {
	v, ok := a.outputs[name]
	return v, ok
}

// TestEvalPercentStatTag_Scales: PercentStat: mult = GetStat(stat) * percent/100; value = value * mult.
func TestEvalPercentStatTag_Scales(t *testing.T) {
	db := NewModDB()
	db.Actor = stubActor{outputs: map[string]float64{"Mana": 200}}

	m := mod.NewFloat("ManaDegen", mod.TypeBase, 1).Tag(mod.PercentStat("Mana", 5))
	db.AddMod(m)

	first := db.Sum(mod.TypeBase, nil, "ManaDegen")
	second := db.Sum(mod.TypeBase, nil, "ManaDegen")

	testza.AssertEqual(t, float64(10), first)
	testza.AssertEqual(t, second, first)
	testza.AssertEqual(t, float64(1), m.Value().Float())
}

// TestEvalDistanceRampTag_Scales: DistanceRamp: value scaled by the ramp step matching cfg.SkillDist.
func TestEvalDistanceRampTag_Scales(t *testing.T) {
	ramp := [][]float64{{35, 0}, {70, 1}}

	cases := []struct {
		name string
		dist float64
		want float64
	}{
		{"at min clamps to first", 35, 0},
		{"at max clamps to last", 70, 50},
		{"interpolates midpoint", 52.5, 25},
		{"below min clamps low", 10, 0},
		{"above max clamps high", 200, 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := NewModDB()
			m := mod.NewFloat("Damage", mod.TypeIncrease, 50).Tag(mod.DistanceRamp(ramp))
			db.AddMod(m)
			cfg := &ListCfg{SkillDist: tc.dist, SkillDistSet: true}
			testza.AssertEqual(t, tc.want, db.Sum(mod.TypeIncrease, cfg, "Damage"))
			testza.AssertEqual(t, float64(50), m.Value().Float())
		})
	}
}

// TestEvalDistanceRampTag_NoDistanceExcludes: DistanceRamp with no distance (nil cfg or unset SkillDist) excludes the mod (PoB `return`).
func TestEvalDistanceRampTag_NoDistanceExcludes(t *testing.T) {
	ramp := [][]float64{{35, 0}, {70, 1}}
	db := NewModDB()
	db.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 50).Tag(mod.DistanceRamp(ramp)))

	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeIncrease, nil, "Damage"))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeIncrease, &ListCfg{SkillDist: 0}, "Damage"))
}

// TestEvalMeleeProximityTag_Scales: MeleeProximity: value scaled by proximity ramp {MaxBonusPct, MinBonusPct}.
func TestEvalMeleeProximityTag_Scales(t *testing.T) {
	cases := []struct {
		name string
		dist float64
		want float64
	}{
		{"max potency <=15", 10, 50},
		{"linear falloff", 27.5, 25},
		{"zero at >=40", 40, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := NewModDB()
			m := mod.NewFloat("Damage", mod.TypeMore, 50).Tag(mod.MeleeProximity([]int{1, 0}))
			db.AddMod(m)
			cfg := &ListCfg{SkillDist: tc.dist, SkillDistSet: true}
			testza.AssertEqual(t, tc.want, db.Sum(mod.TypeMore, cfg, "Damage"))
			testza.AssertEqual(t, float64(50), m.Value().Float())
		})
	}
}

func TestEvalMeleeProximityTag_NoDistanceExcludes(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Damage", mod.TypeMore, 50).Tag(mod.MeleeProximity([]int{1, 0})))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeMore, nil, "Damage"))
}

// TestEvalMeleeProximityTag_ZeroDistanceIsFullPotency: a set skillDist of 0 is truthy in Lua (ModStore.lua:576), so it takes the <= 15 arm.
func TestEvalMeleeProximityTag_ZeroDistanceIsFullPotency(t *testing.T) {
	db := NewModDB()
	db.AddMod(mod.NewFloat("Damage", mod.TypeMore, 50).Tag(mod.MeleeProximity([]int{1, 0})))
	testza.AssertEqual(t, float64(0), db.Sum(mod.TypeMore, &ListCfg{SkillDist: 0}, "Damage"))
	testza.AssertEqual(t, float64(50), db.Sum(mod.TypeMore, &ListCfg{SkillDist: 0, SkillDistSet: true}, "Damage"))
}
