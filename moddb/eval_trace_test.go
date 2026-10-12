package moddb

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob/mod"
)

// TestTraceEvalModAgreesWithEvalMod: TraceEvalMod is the diag's whole claim to being trustworthy.
func TestTraceEvalModAgreesWithEvalMod(t *testing.T) {
	db := NewModDB()
	db.Multipliers["A"] = 3
	db.Multipliers["B"] = 2
	db.Conditions["Met"] = true

	cfg := &ListCfg{SkillStats: map[string]float64{"Strength": 200}, SkillPart: 2}

	cases := map[string]mod.Mod{
		"untagged":             mod.NewFloat("Stat", mod.TypeBase, 100),
		"scaled":               mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Multiplier("A")),
		"threaded scaling":     mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Multiplier("A"), mod.Multiplier("B")),
		"gate passes":          mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Condition("Met")),
		"gate fails":           mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Condition("Unmet")),
		"gate before scale":    mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Condition("Unmet"), mod.Multiplier("A")),
		"scale before gate":    mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Multiplier("A"), mod.Condition("Unmet")),
		"percent stat":         mod.NewFloat("Stat", mod.TypeBase, 1).Tag(mod.PercentStat("Strength", 50)),
		"skill part matches":   mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.SkillPart(2)),
		"skill part rejects":   mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.SkillPart(3)),
		"absent actor drops":   mod.NewFloat("Stat", mod.TypeBase, 100).Tag(mod.Multiplier("A").Actor("enemy")),
		"flag value":           mod.NewFlag("Stat", true).Tag(mod.Condition("Met")),
		"flag value gated out": mod.NewFlag("Stat", true).Tag(mod.Condition("Unmet")),
	}

	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			want := db.evalMod(m, cfg)
			got := TraceEvalMod(db, m, cfg)

			testza.AssertEqual(t, want == nil, got.Excluded())
			if want == nil {
				return
			}
			testza.AssertEqual(t, want.Type(), got.Final.Type())
			testza.AssertEqual(t, want.Float(), got.Final.Float())
			testza.AssertEqual(t, want.Flag(), got.Final.Flag())
			testza.AssertEqual(t, len(m.Tags()), len(got.Steps))
		})
	}
}

// TestTraceEvalModStopsAtTheGate: A gated mod's trace must STOP at the tag that gated it, so the report names one culprit rather than every tag after it.
func TestTraceEvalModStopsAtTheGate(t *testing.T) {
	db := NewModDB()
	db.Multipliers["A"] = 3

	m := mod.NewFloat("Stat", mod.TypeBase, 100).
		Tag(mod.Multiplier("A"), mod.Condition("Unmet"), mod.Multiplier("A"))

	got := TraceEvalMod(db, m, cfgNil())
	testza.AssertTrue(t, got.Excluded())
	testza.AssertEqual(t, 2, len(got.Steps))
	testza.AssertEqual(t, float64(300), got.Steps[1].Before.Float())
	testza.AssertTrue(t, got.Steps[1].After == nil)
	testza.AssertEqual(t, float64(100), m.Value().Float())
}

// TestCfgAdmitsMatchesSum: CfgAdmits must agree with the predicate the aggregate queries apply.
func TestCfgAdmitsMatchesSum(t *testing.T) {
	attack := mod.MFlagAttack
	cfg := &ListCfg{Flags: &attack}

	admitted := mod.NewFloat("Stat", mod.TypeBase, 10).Flag(mod.MFlagAttack)
	rejected := mod.NewFloat("Stat", mod.TypeBase, 10).Flag(mod.MFlagSpell)

	db := NewModDB()
	db.AddMod(admitted)
	db.AddMod(rejected)

	a, b, c := CfgAdmits(admitted, cfg)
	testza.AssertTrue(t, a && b && c)
	a, b, c = CfgAdmits(rejected, cfg)
	testza.AssertFalse(t, a && b && c)

	testza.AssertEqual(t, float64(10), db.Sum(mod.TypeBase, cfg, "Stat"))
}

func cfgNil() *ListCfg { return nil }
